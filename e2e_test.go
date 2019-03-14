package typhon

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/fortytw2/leaktest"
	"github.com/monzo/terrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
)

type e2eFlavour interface {
	Serve(Service) *Server
	URL(*Server) string
	Proto() string
}

// flavours runs the passed E2E test with all test flavours (HTTP/1.1, HTTP/2.0/h2c, etc.)
func flavours(t *testing.T, impl func(*testing.T, e2eFlavour)) {
	someFlavours(t, nil, impl)
}

// someFlavours runs the passed test with only the passed flavours
func someFlavours(t *testing.T, only []string, impl func(*testing.T, e2eFlavour)) {
	run := func(name string) bool {
		if only == nil {
			return true
		}
		for _, o := range only {
			if name == o {
				return true
			}
		}
		return false
	}

	onlys := make(map[string]bool, len(only))
	for _, o := range only {
		onlys[o] = true
	}

	if run("http1.1") {
		t.Run("http1.1", func(t *testing.T) {
			defer leaktest.Check(t)()
			Client = Service(BareClient).Filter(ErrorFilter)
			impl(t, http1Flavour{
				T: t})
		})
	}
	if run("http1.1-tls") {
		t.Run("http1.1-tls", func(t *testing.T) {
			defer leaktest.Check(t)()
			cert := keypair(t, []string{"localhost"})
			Client = HttpService(&http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true},
			}).Filter(ErrorFilter)
			impl(t, http1TLSFlavour{
				T:    t,
				cert: cert})
		})
	}
	if run("http2.0-h2c") {
		t.Run("http2.0-h2c", func(t *testing.T) {
			defer leaktest.Check(t)()
			transport := &http2.Transport{
				AllowHTTP: true,
				DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
					return net.Dial(network, addr)
				}}
			Client = HttpService(transport).Filter(ErrorFilter)
			impl(t, http2H2cFlavour{T: t})
		})
	}
	if run("http2.0-h2") {
		t.Run("http2.0-h2", func(t *testing.T) {
			defer leaktest.Check(t)()
			cert := keypair(t, []string{"localhost"})
			transport := &http2.Transport{
				AllowHTTP: false,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true}}
			Client = HttpService(transport).Filter(ErrorFilter)
			impl(t, http2H2Flavour{
				T:    t,
				cert: cert})
		})
	}
}

func TestE2E(t *testing.T) {
	flavours(t, func(t *testing.T, flav e2eFlavour) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		svc := Service(func(req Request) Response {
			// Simple requests like this shouldn't be chunked
			assert.NotContains(t, req.TransferEncoding, "chunked")
			assert.True(t, req.ContentLength > 0)
			return req.Response(map[string]string{
				"b": "a"})
		})
		svc = svc.Filter(ErrorFilter)
		s := flav.Serve(svc)
		defer s.Stop(context.Background())

		req := NewRequest(ctx, "GET", flav.URL(s), map[string]string{
			"a": "b"})
		rsp := req.Send().Response()
		require.NoError(t, rsp.Error)
		assert.Equal(t, http.StatusOK, rsp.StatusCode)
		assert.Equal(t, flav.Proto(), rsp.Proto)
		require.NotNil(t, rsp.Request)
		assert.Equal(t, req, *rsp.Request)
		body := map[string]string{}
		require.NoError(t, rsp.Decode(&body))
		assert.Equal(t, map[string]string{
			"b": "a"}, body)
		// The response is simple too; shouldn't be chunked
		assert.NotContains(t, rsp.TransferEncoding, "chunked")
		assert.EqualValues(t, 10, rsp.ContentLength)
	})
}

func TestE2EStreaming(t *testing.T) {
	someFlavours(t, []string{"http1.1", "http1.1-tls"}, func(t *testing.T, flav e2eFlavour) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		chunks := make(chan []byte)
		svc := Service(func(req Request) Response {
			rsp := req.Response(nil)
			s := Streamer()
			go func() {
				defer s.Close()
				for c := range chunks {
					_, err := s.Write(c)
					require.NoError(t, err)
				}
			}()
			rsp.Body = s
			return rsp
		})
		svc = svc.Filter(ErrorFilter)
		s := flav.Serve(svc)
		defer s.Stop(context.Background())

		req := NewRequest(ctx, "GET", flav.URL(s), nil)
		rsp := req.Send().Response()
		require.NoError(t, rsp.Error)
		assert.Equal(t, http.StatusOK, rsp.StatusCode)

		for i := 0; i < 10; i++ {
			v := fmt.Sprintf("w®ï†é %d", i)
			chunks <- []byte(v)
			b := make([]byte, len(v)*2)
			n, err := rsp.Body.Read(b)
			require.NoError(t, err)
			assert.Equal(t, v, string(b[:n]))
		}
		close(chunks)
		_, err := rsp.Body.Read(make([]byte, 100))
		assert.Equal(t, io.EOF, err)
	})

	// The HTTP/2.0 streaming implementation is more advanced, as it allows the response body to be streamed back
	// concurrently with the request body. This test constructs a server that echoes the request body back to the client
	// and asserts that the chunks are returned in real time.
	someFlavours(t, []string{"http2.0-h2", "http2.0-h2c"}, func(t *testing.T, flav e2eFlavour) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		svc := Service(func(req Request) Response {
			rsp := req.Response(nil)
			rsp.Body = req.Body
			return rsp
		})
		svc = svc.Filter(ErrorFilter)
		s := flav.Serve(svc)
		defer s.Stop(context.Background())

		req := NewRequest(ctx, "GET", flav.URL(s), nil)
		reqS := Streamer()
		req.Body = reqS
		rsp := req.Send().Response()
		require.NoError(t, rsp.Error)
		assert.Equal(t, http.StatusOK, rsp.StatusCode)

		for i := 0; i < 10; i++ {
			v := fmt.Sprintf("w®ï†é %d", i)
			_, err := io.WriteString(reqS, v)
			require.NoError(t, err)
			b := make([]byte, len(v)*2)
			n, err := rsp.Body.Read(b)
			require.NoError(t, err)
			assert.Equal(t, v, string(b[:n]))
		}
		reqS.Close()
		_, err := rsp.Body.Read(make([]byte, 100))
		assert.Equal(t, io.EOF, err)
	})
}

func TestE2EDomainSocket(t *testing.T) {
	someFlavours(t, []string{"http1.1"}, func(t *testing.T, flav e2eFlavour) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		svc := Service(func(req Request) Response {
			return NewResponse(req)
		})
		svc = svc.Filter(ErrorFilter)

		addr := &net.UnixAddr{
			Net:  "unix",
			Name: "/tmp/typhon-test.sock"}
		l, err := net.ListenUnix("unix", addr)
		require.NoError(t, err)
		defer l.Close()

		s, err := Serve(svc, l)
		require.NoError(t, err)
		defer s.Stop(context.Background())

		sockTransport := &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.DialUnix("unix", nil, addr)
			}}
		req := NewRequest(ctx, "GET", "http://localhost/foo", nil)
		rsp := req.SendVia(HttpService(sockTransport)).Response()
		require.NoError(t, rsp.Error)
		assert.Equal(t, http.StatusOK, rsp.StatusCode)
		assert.Equal(t, flav.Proto(), rsp.Proto)
	})
}

func TestE2EError(t *testing.T) {
	flavours(t, func(t *testing.T, flav e2eFlavour) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		expectedErr := terrors.Unauthorized("ah_ah_ah", "You didn't say the magic word!", map[string]string{
			"param": "value"})
		svc := Service(func(req Request) Response {
			rsp := Response{
				Error: expectedErr}
			rsp.Write([]byte("throwaway")) // should be removed
			return rsp
		})
		svc = svc.Filter(ErrorFilter)
		s := flav.Serve(svc)
		defer s.Stop(context.Background())

		req := NewRequest(ctx, "GET", flav.URL(s), nil)
		rsp := req.Send().Response()
		assert.Equal(t, http.StatusUnauthorized, rsp.StatusCode)
		assert.True(t, rsp.ContentLength > 0)

		b, _ := rsp.BodyBytes(false)
		assert.NotContains(t, string(b), "throwaway")

		require.Error(t, rsp.Error)
		terr := terrors.Wrap(rsp.Error, nil).(*terrors.Error)
		terrExpect := terrors.Unauthorized("ah_ah_ah", "You didn't say the magic word!", nil)
		assert.Equal(t, terrExpect.Message, terr.Message)
		assert.Equal(t, terrExpect.Code, terr.Code)
		assert.Equal(t, "value", terr.Params["param"])
	})
}

func TestE2ECancellation(t *testing.T) {
	flavours(t, func(t *testing.T, flav e2eFlavour) {
		cancelled := make(chan struct{})
		svc := Service(func(req Request) Response {
			<-req.Done()
			close(cancelled)
			return req.Response("cancelled ok")
		})
		svc = svc.Filter(ErrorFilter)
		s := flav.Serve(svc)
		defer s.Stop(context.Background())

		ctx, cancel := context.WithCancel(context.Background())
		req := NewRequest(ctx, "GET", flav.URL(s), nil)
		req.Send()
		select {
		case <-cancelled:
			assert.Fail(t, "cancellation propagated prematurely")
		case <-time.After(30 * time.Millisecond):
		}
		cancel()
		select {
		case <-cancelled:
		case <-time.After(30 * time.Millisecond):
			assert.Fail(t, "cancellation not propagated")
		}
	})
}

func TestE2ENoFollowRedirect(t *testing.T) {
	flavours(t, func(t *testing.T, flav e2eFlavour) {
		svc := Service(func(req Request) Response {
			if req.URL.Path == "/redirected" {
				return req.Response("😱")
			}

			rsp := req.Response(nil)
			dst := fmt.Sprintf("http://%s/redirected", req.Host)
			http.Redirect(rsp.Writer(), &req.Request, dst, http.StatusFound)
			return rsp
		})
		s := flav.Serve(svc)
		defer s.Stop(context.Background())

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		req := NewRequest(ctx, "GET", flav.URL(s), nil)
		rsp := req.Send().Response()
		assert.NoError(t, rsp.Error)
		assert.Equal(t, http.StatusFound, rsp.StatusCode)
		assert.EqualValues(t, 56, rsp.ContentLength)
	})
}

func TestE2EProxiedStreamer(t *testing.T) {
	flavours(t, func(t *testing.T, flav e2eFlavour) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		chunks := make(chan bool)
		downstream := Service(func(req Request) Response {
			rsp := req.Response(nil)
			rsp.Body = Streamer()
			go func() {
				defer rsp.Body.Close()
				n := 0
				for range chunks {
					rsp.Encode(map[string]int{
						"chunk": n})
					n++
				}
			}()
			return rsp
		})
		s := flav.Serve(downstream)
		defer s.Stop(context.Background())

		proxy := Service(func(req Request) Response {
			proxyReq := NewRequest(req, "GET", flav.URL(s), nil)
			return proxyReq.Send().Response()
		})
		ps := flav.Serve(proxy)
		defer ps.Stop(context.Background())

		req := NewRequest(ctx, "GET", flav.URL(ps), nil)
		rsp := req.Send().Response()
		assert.NoError(t, rsp.Error)
		assert.Equal(t, http.StatusOK, rsp.StatusCode)
		if !rsp.ProtoAtLeast(2, 0) {
			assert.Contains(t, rsp.TransferEncoding, "chunked")
		}
		assert.EqualValues(t, -1, rsp.ContentLength)
		for i := 0; i < 100; i++ {
			chunks <- true
			b := make([]byte, 500)
			n, err := rsp.Body.Read(b)
			require.NoError(t, err)
			v := map[string]int{}
			require.NoError(t, json.Unmarshal(b[:n], &v))
			require.Equal(t, i, v["chunk"])
		}
		close(chunks)
		_, err := rsp.Body.Read(make([]byte, 100))
		assert.Equal(t, io.EOF, err)
	})
}

// TestE2EInfiniteContext verifies that Typhon does not leak Goroutines if an infinite context (one that's never
// cancelled) is used to make a request.
func TestE2EInfiniteContext(t *testing.T) {
	flavours(t, func(t *testing.T, flav e2eFlavour) {
		ctx := context.Background()

		var receivedCtx context.Context
		svc := Service(func(req Request) Response {
			receivedCtx = req.Context
			return req.Response(map[string]string{
				"b": "a"})
		})
		svc = svc.Filter(ErrorFilter)
		s := flav.Serve(svc)
		defer s.Stop(context.Background())

		req := NewRequest(ctx, "GET", flav.URL(s), map[string]string{
			"a": "b"})
		rsp := req.Send().Response()
		require.NoError(t, rsp.Error)
		assert.Equal(t, http.StatusOK, rsp.StatusCode)

		b, err := ioutil.ReadAll(rsp.Body)
		require.NoError(t, err)
		assert.Equal(t, "{\"b\":\"a\"}\n", string(b))

		// Consuming the body should have closed the receiving context
		select {
		case <-receivedCtx.Done():
		case <-time.After(time.Second):
			assert.Fail(t, "cancellation not propagated")
		}
	})
}

func TestE2ERequestAutoChunking(t *testing.T) {
	someFlavours(t, []string{"http1.1", "http1.1-tls"}, func(t *testing.T, flav e2eFlavour) {
		receivedChunked := false
		svc := Service(func(req Request) Response {
			receivedChunked = false
			for _, e := range req.TransferEncoding {
				if e == "chunked" {
					receivedChunked = true
				}
			}
			return req.Response("ok")
		})
		svc = svc.Filter(ErrorFilter)
		s := flav.Serve(svc)
		defer s.Stop(context.Background())

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Streamer; should be chunked
		stream := Streamer()
		go func() {
			io.WriteString(stream, "foo\n")
			stream.Close()
		}()
		req := NewRequest(ctx, "GET", flav.URL(s), nil)
		req.Body = stream
		rsp := req.Send().Response()
		require.NoError(t, rsp.Error)
		assert.Equal(t, http.StatusOK, rsp.StatusCode)
		assert.True(t, receivedChunked)

		// Small request using Encode(): should not be chunked
		req = NewRequest(ctx, "GET", flav.URL(s), map[string]string{
			"a": "b"})
		rsp = req.Send().Response()
		require.NoError(t, rsp.Error)
		assert.Equal(t, http.StatusOK, rsp.StatusCode)
		assert.EqualValues(t, 5, rsp.ContentLength)
		assert.False(t, receivedChunked)

		// Large request using Encode(); should be chunked
		const targetBytes = 5000000 // 5 MB
		body := []byte{}
		for len(body) < targetBytes {
			body = append(body, []byte("abc=def\n")...)
		}
		req = NewRequest(ctx, "GET", flav.URL(s), body)
		rsp = req.Send().Response()
		require.NoError(t, rsp.Error)
		assert.Equal(t, http.StatusOK, rsp.StatusCode)
		assert.True(t, receivedChunked)
	})
}

func TestE2EResponseAutoChunking(t *testing.T) {
	someFlavours(t, []string{"http1.1", "http1.1-tls"}, func(t *testing.T, flav e2eFlavour) {
		var sendRsp Response
		svc := Service(func(req Request) Response {
			sendRsp.Request = &req
			return sendRsp
		})
		svc = svc.Filter(ErrorFilter)
		s := flav.Serve(svc)
		defer s.Stop(context.Background())

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Streamer; should be chunked
		req := NewRequest(ctx, "GET", flav.URL(s), nil)
		sendRsp = NewResponse(req)
		stream := Streamer()
		go func() {
			io.WriteString(stream, "foo\n")
			stream.Close()
		}()
		sendRsp.Body = stream
		rsp := req.Send().Response()
		require.NoError(t, rsp.Error)
		assert.Equal(t, http.StatusOK, rsp.StatusCode)
		assert.Contains(t, rsp.TransferEncoding, "chunked")
		assert.EqualValues(t, -1, rsp.ContentLength)

		// Small request using Encode(): should not be chunked
		sendRsp = NewResponse(req)
		sendRsp.Encode(map[string]string{
			"a": "b"})
		rsp = req.Send().Response()
		require.NoError(t, rsp.Error)
		assert.Equal(t, http.StatusOK, rsp.StatusCode)
		assert.EqualValues(t, 10, rsp.ContentLength)
		assert.NotContains(t, rsp.TransferEncoding, "chunked")

		// Large request using Encode(); should be chunked
		const targetBytes = 5000000 // 5 MB
		body := []byte{}
		for len(body) < targetBytes {
			body = append(body, []byte("abc=def\n")...)
		}
		sendRsp = NewResponse(req)
		sendRsp.Encode(body)
		rsp = req.Send().Response()
		require.NoError(t, rsp.Error)
		assert.Equal(t, http.StatusOK, rsp.StatusCode)
		assert.EqualValues(t, -1, rsp.ContentLength)
		assert.Contains(t, rsp.TransferEncoding, "chunked")
	})
}

// TestStreamingCancellation asserts that a server's writes won't block forever if a client cancels a request
func TestE2EStreamingCancellation(t *testing.T) {
	flavours(t, func(t *testing.T, flav e2eFlavour) {
		done := make(chan struct{})
		svc := Service(func(req Request) Response {
			s := Streamer()
			go func() {
				defer close(done)
				io.WriteString(s, "derp\n")
				<-req.Done()
				// Write a bunch of chunks; this should not block forever even though the client has gone
				for i := 0; i < 500; i++ {
					io.WriteString(s, "derp\n")
				}
			}()
			rsp := req.Response(nil)
			rsp.Body = s
			return rsp
		})
		svc = svc.Filter(ErrorFilter)
		s := flav.Serve(svc)
		defer s.Stop(context.Background())

		ctx, cancel := context.WithCancel(context.Background())
		req := NewRequest(ctx, "GET", flav.URL(s), nil)
		req.Send().Response()
		cancel()
		<-done
	})
}

// TestE2EFullDuplex verifies that HTTP/2.0 full-duplex communication works properly. It constructs a service which
// will write chunks of output by both copying them from the request body, and writing them directly. The two should
// be interleaved and sent to the client without delay.
func TestE2EFullDuplex(t *testing.T) {
	someFlavours(t, []string{"http2.0-h2", "http2.0-h2c"}, func(t *testing.T, flav e2eFlavour) {
		chunks := make(chan []byte)
		svc := Service(func(req Request) Response {
			body := Streamer()
			go func() {
				defer body.Close()
				go io.Copy(body, req.Body)
				for c := range chunks {
					body.Write(c)
				}
			}()
			return req.Response(body)
		})
		svc = svc.Filter(ErrorFilter)
		s := flav.Serve(svc)
		defer s.Stop(context.Background())

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		req := NewRequest(ctx, "GET", flav.URL(s), nil)
		req.Body = Streamer()
		defer req.Body.Close()
		rsp := req.Send().Response()
		assert.EqualValues(t, -1, rsp.ContentLength)

		for i := 0; i < 50; i++ {
			b := []byte(fmt.Sprintf("foo %d", i))
			if i%2 == 0 { // Alternate between sending a chunk in the request body, and sending it "directly"
				req.Write(b)
			} else {
				chunks <- b
			}
			bb := make([]byte, len(b)*2)
			n, _ := rsp.Body.Read(bb)
			bb = bb[:n]
			assert.Equal(t, b, bb)
		}
		close(chunks)

		assert.EqualValues(t, -1, rsp.ContentLength)
	})
}

func TestE2EDraining(t *testing.T) {
	flavours(t, func(t *testing.T, flav e2eFlavour) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		returnRsp := make(chan bool)
		svc := Service(func(req Request) Response {
			<-returnRsp
			return NewResponse(req)
		})
		svc = svc.Filter(ErrorFilter)
		s := flav.Serve(svc)

		// Send a request, which will hang in the handler until we send to returnRsp
		req := NewRequest(ctx, "GET", flav.URL(s), nil)
		rspF := req.Send()
		time.Sleep(10 * time.Millisecond) // allow connection to be established

		// Stop the server; the in-flight request should remain pending and the server should sit in graceful shutdown
		// until the request completes
		serverClosed := make(chan struct{})
		go func() {
			s.Stop(ctx)
			close(serverClosed)
		}()
		select {
		case <-serverClosed:
			require.FailNow(t, "server closed with request outstanding")
		case <-rspF.WaitC():
			require.FailNow(t, "premature response")
		case <-time.After(100 * time.Millisecond):
		}

		// Send a new request (while the original one is still in-flight); this one should be rejected
		req2 := NewRequest(ctx, "GET", flav.URL(s), nil)
		rsp2 := req2.Send().Response()
		require.Error(t, rsp2.Error)

		// Unblock the handler; a successful response should be returned and server shutdown should complete
		returnRsp <- true
		rsp := rspF.Response()
		require.NoError(t, rsp.Error)
		<-serverClosed
	})
}
