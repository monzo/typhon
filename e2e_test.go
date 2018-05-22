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
	"github.com/stretchr/testify/suite"
)

func TestE2E(t *testing.T) {
	t.Parallel()
	suite.Run(t, &e2eSuite{})
}

type e2eSuite struct {
	suite.Suite
}

func (suite *e2eSuite) SetupTest() {
	Client = Service(BareClient).Filter(ErrorFilter)
}

func (suite *e2eSuite) TearDownTest() {
	Client = BareClient
}

func (suite *e2eSuite) serve(svc Service) Server {
	s, err := Listen(svc, "localhost:0")
	suite.Require().NoError(err)
	return s
}

func (suite *e2eSuite) TestStraightforward() {
	defer leaktest.Check(suite.T())()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc := Service(func(req Request) Response {
		// Simple requests like this shouldn't be chunked
		suite.Assert().NotContains(req.TransferEncoding, "chunked")
		suite.Assert().True(req.ContentLength > 0)
		return req.Response(map[string]string{
			"b": "a"})
	})
	svc = svc.Filter(ErrorFilter)
	s := suite.serve(svc)
	defer s.Stop()

	req := NewRequest(ctx, "GET", fmt.Sprintf("http://%s", s.Listener().Addr()), map[string]string{
		"a": "b"})
	rsp := req.Send().Response()
	suite.Require().NoError(rsp.Error)
	suite.Assert().Equal(http.StatusOK, rsp.StatusCode)
	body := map[string]string{}
	suite.Assert().NoError(rsp.Decode(&body))
	suite.Assert().Equal(map[string]string{
		"b": "a"}, body)
	// The response is simple too; shouldn't be chunked
	suite.Assert().NotContains(rsp.TransferEncoding, "chunked")
	suite.Assert().True(rsp.ContentLength > 0)
}

func (suite *e2eSuite) TestDomainSocket() {
	defer leaktest.Check(suite.T())()
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
	suite.Require().NoError(err)
	defer l.Close()

	s, err := Serve(svc, l)
	suite.Require().NoError(err)
	defer s.Stop()

	sockTransport := &httpcontrol.Transport{
		Dial: func(network, address string) (net.Conn, error) {
			return net.DialUnix("unix", nil, addr)
		}}
	req := NewRequest(ctx, "GET", "http://localhost/foo", nil)
	rsp := req.SendVia(HttpService(sockTransport)).Response()
	suite.Require().NoError(rsp.Error)
	suite.Assert().Equal(http.StatusOK, rsp.StatusCode)
}

func (suite *e2eSuite) TestError() {
	defer leaktest.Check(suite.T())()
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
	s := suite.serve(svc)
	defer s.Stop()

	req := NewRequest(ctx, "GET", fmt.Sprintf("http://%s", s.Listener().Addr()), nil)
	rsp := req.Send().Response()
	suite.Assert().Equal(http.StatusUnauthorized, rsp.StatusCode)

	b, _ := rsp.BodyBytes(false)
	suite.Assert().NotContains(string(b), "throwaway")

	suite.Require().Error(rsp.Error)
	terr := terrors.Wrap(rsp.Error, nil).(*terrors.Error)
	terrExpect := terrors.Unauthorized("ah_ah_ah", "You didn't say the magic word!", nil)
	suite.Assert().Equal(terrExpect.Message, terr.Message)
	suite.Assert().Equal(terrExpect.Code, terr.Code)
	suite.Assert().Equal("value", terr.Params["param"])
}

func (suite *e2eSuite) TestCancellation() {
	defer leaktest.Check(suite.T())()

	cancelled := make(chan struct{})
	svc := Service(func(req Request) Response {
		<-req.Done()
		close(cancelled)
		return req.Response("cancelled ok")
	})
	svc = svc.Filter(ErrorFilter)
	s := suite.serve(svc)
	defer s.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	req := NewRequest(ctx, "GET", fmt.Sprintf("http://%s/", s.Listener().Addr()), nil)
	req.Send()
	select {
	case <-cancelled:
		suite.Assert().Fail("cancellation propagated prematurely")
	case <-time.After(30 * time.Millisecond):
	}
	cancel()
	select {
	case <-cancelled:
	case <-time.After(30 * time.Millisecond):
		suite.Assert().Fail("cancellation not propagated")
	}
}

func (suite *e2eSuite) TestNoFollowRedirect() {
	defer leaktest.Check(suite.T())()
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
	s := suite.serve(svc)
	defer s.Stop()
	req := NewRequest(ctx, "GET", fmt.Sprintf("http://%s/", s.Listener().Addr()), nil)
	rsp := req.Send().Response()
	suite.Assert().NoError(rsp.Error)
	suite.Assert().Equal(http.StatusFound, rsp.StatusCode)
}

func (suite *e2eSuite) TestProxiedStreamer() {
	defer leaktest.Check(suite.T())()
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
	s := suite.serve(downstream)
	defer s.Stop()

	proxy := Service(func(req Request) Response {
		proxyReq := NewRequest(req, "GET", fmt.Sprintf("http://%s/", s.Listener().Addr()), nil)
		return proxyReq.Send().Response()
	})
	ps := suite.serve(proxy)
	defer ps.Stop()

	req := NewRequest(ctx, "GET", fmt.Sprintf("http://%s/", ps.Listener().Addr()), nil)
	rsp := req.Send().Response()
	suite.Assert().NoError(rsp.Error)
	suite.Assert().Equal(http.StatusOK, rsp.StatusCode)
	// The response is streaming; should be chunked
	suite.Assert().Contains(rsp.TransferEncoding, "chunked")
	suite.Assert().True(rsp.ContentLength < 0)
	for i := 0; i < 1000; i++ {
		b := make([]byte, 500)
		n, err := rsp.Body.Read(b)
		suite.Require().NoError(err)
		v := map[string]int{}
		suite.Require().NoError(json.Unmarshal(b[:n], &v))
		suite.Require().Equal(i, v["chunk"])
		chunks <- true
	}
	close(chunks)
}

// TestInfiniteContext verifies that Typhon does not leak Goroutines if an infinite context (one that's never cancelled)
// is used to make a request.
func (suite *e2eSuite) TestInfiniteContext() {
	defer leaktest.Check(suite.T())()
	ctx := context.Background()

	var receivedCtx context.Context
	svc := Service(func(req Request) Response {
		receivedCtx = req.Context
		return req.Response(map[string]string{
			"b": "a"})
	})
	svc = svc.Filter(ErrorFilter)
	s := suite.serve(svc)
	defer s.Stop()

	req := NewRequest(ctx, "GET", fmt.Sprintf("http://%s", s.Listener().Addr()), map[string]string{
		"a": "b"})
	rsp := req.Send().Response()
	suite.Require().NoError(rsp.Error)
	suite.Assert().Equal(http.StatusOK, rsp.StatusCode)

	b, err := ioutil.ReadAll(rsp.Body)
	suite.Require().NoError(err)
	suite.Assert().Equal("{\"b\":\"a\"}\n", string(b))

	// Consuming the body should have closed the receiving context
	select {
	case <-receivedCtx.Done():
	case <-time.After(time.Second):
		suite.Assert().Fail("cancellation not propagated")
	}
}

func (suite *e2eSuite) TestRequestAutoChunking() {
	defer leaktest.Check(suite.T())()
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
	s := suite.serve(svc)
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
	suite.Require().NoError(rsp.Error)
	suite.Assert().Equal(http.StatusOK, rsp.StatusCode)
	suite.Assert().True(receivedChunked)

	// Small request using Encode(): should not be chunked
	req = NewRequest(ctx, "GET", fmt.Sprintf("http://%s", s.Listener().Addr()), map[string]string{
		"a": "b"})
	rsp = req.Send().Response()
	suite.Require().NoError(rsp.Error)
	suite.Assert().Equal(http.StatusOK, rsp.StatusCode)
	suite.Assert().False(receivedChunked)

	// Large request using Encode(); should be chunked
	const targetBytes = 5000000 // 5 MB
	body := []byte{}
	for len(body) < targetBytes {
		body = append(body, []byte("abc=def\n")...)
	}
	req = NewRequest(ctx, "GET", fmt.Sprintf("http://%s", s.Listener().Addr()), body)
	rsp = req.Send().Response()
	suite.Require().NoError(rsp.Error)
	suite.Assert().Equal(http.StatusOK, rsp.StatusCode)
	suite.Assert().True(receivedChunked)
}

func (suite *e2eSuite) TestResponseAutoChunking() {
	defer leaktest.Check(suite.T())()
	var sendRsp Response
	svc := Service(func(req Request) Response {
		sendRsp.ctx = req
		return sendRsp
	})
	svc = svc.Filter(ErrorFilter)
	s := suite.serve(svc)
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
	suite.Require().NoError(rsp.Error)
	suite.Assert().Equal(http.StatusOK, rsp.StatusCode)
	suite.Assert().Contains(rsp.TransferEncoding, "chunked")

	// Small request using Encode(): should not be chunked
	sendRsp = NewResponse(req)
	sendRsp.Encode(map[string]string{
		"a": "b"})
	rsp = req.Send().Response()
	suite.Require().NoError(rsp.Error)
	suite.Assert().Equal(http.StatusOK, rsp.StatusCode)
	suite.Assert().NotContains(rsp.TransferEncoding, "chunked")

	// Large request using Encode(); should be chunked
	const targetBytes = 5000000 // 5 MB
	body := []byte{}
	for len(body) < targetBytes {
		body = append(body, []byte("abc=def\n")...)
	}
	sendRsp = NewResponse(req)
	sendRsp.Encode(body)
	rsp = req.Send().Response()
	suite.Require().NoError(rsp.Error)
	suite.Assert().Equal(http.StatusOK, rsp.StatusCode)
	suite.Assert().Contains(rsp.TransferEncoding, "chunked")
}
