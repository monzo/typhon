package typhon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/facebookgo/httpcontrol"
	"github.com/fortytw2/leaktest"
	"github.com/monzo/terrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E(t *testing.T) {
	defer leaktest.Check(t)()
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
	s := serve(t, svc)
	defer s.Stop()

	req := NewRequest(ctx, "GET", fmt.Sprintf("http://%s", s.Listener().Addr()), map[string]string{
		"a": "b"})
	rsp := req.Send().Response()
	require.NoError(t, rsp.Error)
	assert.Equal(t, http.StatusOK, rsp.StatusCode)
	require.NotNil(t, rsp.Request)
	assert.Equal(t, req, *rsp.Request)
	body := map[string]string{}
	assert.NoError(t, rsp.Decode(&body))
	assert.Equal(t, map[string]string{
		"b": "a"}, body)
	// The response is simple too; shouldn't be chunked
	assert.NotContains(t, rsp.TransferEncoding, "chunked")
	assert.True(t, rsp.ContentLength > 0)
}

func TestE2EDomainSocket(t *testing.T) {
	defer leaktest.Check(t)()
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
	defer s.Stop()

	sockTransport := &httpcontrol.Transport{
		Dial: func(network, address string) (net.Conn, error) {
			return net.DialUnix("unix", nil, addr)
		}}
	req := NewRequest(ctx, "GET", "http://localhost/foo", nil)
	rsp := req.SendVia(HttpService(sockTransport)).Response()
	require.NoError(t, rsp.Error)
	assert.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestE2EError(t *testing.T) {
	defer leaktest.Check(t)()
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
	s := serve(t, svc)
	defer s.Stop()

	req := NewRequest(ctx, "GET", fmt.Sprintf("http://%s", s.Listener().Addr()), nil)
	rsp := req.Send().Response()
	assert.Equal(t, http.StatusUnauthorized, rsp.StatusCode)

	b, _ := rsp.BodyBytes(false)
	assert.NotContains(t, string(b), "throwaway")

	require.Error(t, rsp.Error)
	terr := terrors.Wrap(rsp.Error, nil).(*terrors.Error)
	terrExpect := terrors.Unauthorized("ah_ah_ah", "You didn't say the magic word!", nil)
	assert.Equal(t, terrExpect.Message, terr.Message)
	assert.Equal(t, terrExpect.Code, terr.Code)
	assert.Equal(t, "value", terr.Params["param"])
}

func TestE2ECancellation(t *testing.T) {
	defer leaktest.Check(t)()

	cancelled := make(chan struct{})
	svc := Service(func(req Request) Response {
		<-req.Done()
		close(cancelled)
		return req.Response("cancelled ok")
	})
	svc = svc.Filter(ErrorFilter)
	s := serve(t, svc)
	defer s.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	req := NewRequest(ctx, "GET", fmt.Sprintf("http://%s/", s.Listener().Addr()), nil)
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
}

func TestNoFollowRedirect(t *testing.T) {
	defer leaktest.Check(t)()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc := Service(func(req Request) Response {
		if req.URL.Path == "/redirected" {
			return req.Response("😱")
		}

		rsp := req.Response(nil)
		dst := fmt.Sprintf("http://%s/redirected", req.Host)
		http.Redirect(rsp.Writer(), &req.Request, dst, http.StatusFound)
		return rsp
	})
	s := serve(t, svc)
	defer s.Stop()
	req := NewRequest(ctx, "GET", fmt.Sprintf("http://%s/", s.Listener().Addr()), nil)
	rsp := req.Send().Response()
	assert.NoError(t, rsp.Error)
	assert.Equal(t, http.StatusFound, rsp.StatusCode)
}

func TestProxiedStreamer(t *testing.T) {
	defer leaktest.Check(t)()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chunks := make(chan bool, 2)
	chunks <- true
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
	s := serve(t, downstream)
	defer s.Stop()

	proxy := Service(func(req Request) Response {
		proxyReq := NewRequest(req, "GET", fmt.Sprintf("http://%s/", s.Listener().Addr()), nil)
		return proxyReq.Send().Response()
	})
	ps := serve(t, proxy)
	defer ps.Stop()

	req := NewRequest(ctx, "GET", fmt.Sprintf("http://%s/", ps.Listener().Addr()), nil)
	rsp := req.Send().Response()
	assert.NoError(t, rsp.Error)
	assert.Equal(t, http.StatusOK, rsp.StatusCode)
	// The response is streaming; should be chunked
	assert.Contains(t, rsp.TransferEncoding, "chunked")
	assert.True(t, rsp.ContentLength < 0)
	for i := 0; i < 1000; i++ {
		b := make([]byte, 500)
		n, err := rsp.Body.Read(b)
		require.NoError(t, err)
		v := map[string]int{}
		require.NoError(t, json.Unmarshal(b[:n], &v))
		require.Equal(t, i, v["chunk"])
		chunks <- true
	}
	close(chunks)
}

// TestInfiniteContext verifies that Typhon does not leak Goroutines if an infinite context (one that's never cancelled)
// is used to make a request.
func TestInfiniteContext(t *testing.T) {
	defer leaktest.Check(t)()
	ctx := context.Background()

	var receivedCtx context.Context
	svc := Service(func(req Request) Response {
		receivedCtx = req.Context
		return req.Response(map[string]string{
			"b": "a"})
	})
	svc = svc.Filter(ErrorFilter)
	s := serve(t, svc)
	defer s.Stop()

	req := NewRequest(ctx, "GET", fmt.Sprintf("http://%s", s.Listener().Addr()), map[string]string{
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
}

func TestRequestAutoChunking(t *testing.T) {
	defer leaktest.Check(t)()
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
	s := serve(t, svc)
	defer s.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Streamer; should be chunked
	stream := Streamer()
	go func() {
		io.WriteString(stream, "foo\n")
		stream.Close()
	}()
	req := NewRequest(ctx, "GET", fmt.Sprintf("http://%s", s.Listener().Addr()), nil)
	req.Body = stream
	rsp := req.Send().Response()
	require.NoError(t, rsp.Error)
	assert.Equal(t, http.StatusOK, rsp.StatusCode)
	assert.True(t, receivedChunked)

	// Small request using Encode(): should not be chunked
	req = NewRequest(ctx, "GET", fmt.Sprintf("http://%s", s.Listener().Addr()), map[string]string{
		"a": "b"})
	rsp = req.Send().Response()
	require.NoError(t, rsp.Error)
	assert.Equal(t, http.StatusOK, rsp.StatusCode)
	assert.False(t, receivedChunked)

	// Large request using Encode(); should be chunked
	const targetBytes = 5000000 // 5 MB
	body := []byte{}
	for len(body) < targetBytes {
		body = append(body, []byte("abc=def\n")...)
	}
	req = NewRequest(ctx, "GET", fmt.Sprintf("http://%s", s.Listener().Addr()), body)
	rsp = req.Send().Response()
	require.NoError(t, rsp.Error)
	assert.Equal(t, http.StatusOK, rsp.StatusCode)
	assert.True(t, receivedChunked)
}

func TestResponseAutoChunking(t *testing.T) {
	defer leaktest.Check(t)()
	var sendRsp Response
	svc := Service(func(req Request) Response {
		sendRsp.Request = &req
		return sendRsp
	})
	svc = svc.Filter(ErrorFilter)
	s := serve(t, svc)
	defer s.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Streamer; should be chunked
	req := NewRequest(ctx, "GET", fmt.Sprintf("http://%s", s.Listener().Addr()), nil)
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

	// Small request using Encode(): should not be chunked
	sendRsp = NewResponse(req)
	sendRsp.Encode(map[string]string{
		"a": "b"})
	rsp = req.Send().Response()
	require.NoError(t, rsp.Error)
	assert.Equal(t, http.StatusOK, rsp.StatusCode)
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
	assert.Contains(t, rsp.TransferEncoding, "chunked")
}

// TestStreamingCancellation asserts that a server's writes won't block forever if a client cancels a request
func TestStreamingCancellation(t *testing.T) {
	defer leaktest.Check(t)()

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
	s := serve(t, svc)
	defer s.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	req := NewRequest(ctx, "GET", fmt.Sprintf("http://%s/", s.Listener().Addr()), nil)
	req.Send().Response()
	cancel()
	<-done
}

func BenchmarkRequestResponse(b *testing.B) {
	b.ReportAllocs()
	svc := Service(func(req Request) Response {
		rsp := req.Response(nil)
		rsp.Header.Set("a", "b")
		rsp.Header.Set("b", "b")
		rsp.Header.Set("c", "b")
		return rsp
	})
	addr := &net.UnixAddr{
		Net:  "unix",
		Name: "/tmp/typhon-test.sock"}
	l, _ := net.ListenUnix("unix", addr)
	defer l.Close()
	s, _ := Serve(svc, l)
	defer s.Stop()

	sockTransport := &httpcontrol.Transport{
		Dial: func(network, address string) (net.Conn, error) {
			return net.DialUnix("unix", nil, addr)
		}}

	ctx := context.Background()
	req := NewRequest(ctx, "GET", "http://localhost/foo", nil)
	sockSvc := HttpService(sockTransport)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req.SendVia(sockSvc).Response()
	}
}
