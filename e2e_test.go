package typhon

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/facebookgo/httpcontrol"
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

	req := NewRequest(nil, "GET", fmt.Sprintf("http://%s", s.Listener().Addr()), map[string]string{
		"a": "b"})
	rsp := req.Send().Response()
	suite.Require().NoError(rsp.Error)
	suite.Assert().Equal(http.StatusOK, rsp.StatusCode)
	// The response is simple too; shouldn't be chunked
	suite.Assert().NotContains(rsp.TransferEncoding, "chunked")
	suite.Assert().True(rsp.ContentLength > 0)
}

func (suite *e2eSuite) TestDomainSocket() {
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
	req := NewRequest(nil, "GET", "http://localhost/foo", nil)
	rsp := req.SendVia(HttpService(sockTransport)).Response()
	suite.Require().NoError(rsp.Error)
	suite.Assert().Equal(http.StatusOK, rsp.StatusCode)
}

func (suite *e2eSuite) TestError() {
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

	req := NewRequest(nil, "GET", fmt.Sprintf("http://%s", s.Listener().Addr()), nil)
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
	cancelled := make(chan struct{})
	svc := Service(func(req Request) Response {
		<-req.Done()
		close(cancelled)
		return req.Response("cancelled ok")
	})
	svc = svc.Filter(ErrorFilter)
	s := suite.serve(svc)
	defer s.Stop()

	req := NewRequest(nil, "GET", fmt.Sprintf("http://%s/", s.Listener().Addr()), nil)
	f := req.Send()
	select {
	case <-cancelled:
		suite.Assert().Fail("cancellation propagated prematurely")
	case <-time.After(30 * time.Millisecond):
	}
	f.Cancel()
	select {
	case <-cancelled:
	case <-time.After(30 * time.Millisecond):
		suite.Assert().Fail("cancellation not propagated")
	}
}

func (suite *e2eSuite) TestNoFollowRedirect() {
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
	req := NewRequest(nil, "GET", fmt.Sprintf("http://%s/", s.Listener().Addr()), nil)
	rsp := req.Send().Response()
	suite.Assert().NoError(rsp.Error)
	suite.Assert().Equal(http.StatusFound, rsp.StatusCode)
}

func (suite *e2eSuite) TestProxiedStreamer() {
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
		proxyReq := NewRequest(nil, "GET", fmt.Sprintf("http://%s/", s.Listener().Addr()), nil)
		return proxyReq.Send().Response()
	})
	ps := suite.serve(proxy)
	defer ps.Stop()

	req := NewRequest(nil, "GET", fmt.Sprintf("http://%s/", ps.Listener().Addr()), nil)
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
