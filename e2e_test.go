package typhon

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/facebookgo/httpcontrol"
	"github.com/monzo/terrors"
	"github.com/stretchr/testify/suite"
)

func TestE2E(t *testing.T) {
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

func (suite *e2eSuite) TestStraightforward() {
	svc := Service(func(req Request) Response {
		return NewResponse(req)
	})
	svc = svc.Filter(ErrorFilter)
	s, err := Listen(svc, "localhost:30001")
	suite.Require().NoError(err)
	defer s.Stop()

	req := NewRequest(nil, "GET", "http://localhost:30001", nil)
	rsp := req.Send().Response()
	suite.Require().NoError(rsp.Error)
	suite.Assert().Equal(http.StatusOK, rsp.StatusCode)
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
	rsp := req.SendVia(HttpService(&http.Client{
		Transport: sockTransport})).Response()
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
	s, err := Listen(svc, "localhost:30001")
	suite.Require().NoError(err)
	defer s.Stop()

	req := NewRequest(nil, "GET", "http://localhost:30001", nil)
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
		select {
		case <-req.Context.Done():
			close(cancelled)
			return req.Response("ok")
		case <-time.After(3 * time.Second):
			rsp := req.Response("timed out")
			rsp.StatusCode = http.StatusRequestTimeout
			return rsp
		}
	})
	svc = svc.Filter(ErrorFilter)
	s, err := Listen(svc, "localhost:30001")
	suite.Require().NoError(err)
	defer s.Stop()

	req := NewRequest(nil, "GET", "http://localhost:30001", nil)
	f := req.Send()
	time.Sleep(50 * time.Millisecond)
	f.Cancel()
	select {
	case <-cancelled:
	case <-time.After(100 * time.Millisecond):
		suite.Assert().Fail("Did not cancel")
	}
}

func (suite *e2eSuite) TestPreCancellation() {
	ctx, ccl := context.WithCancel(context.Background())
	ccl()
	req := NewRequest(ctx, "GET", "http://localhost", nil)
	rsp := req.Send().Response()

	suite.Require().Error(rsp.Error)
	terr := terrors.Wrap(rsp.Error, nil).(*terrors.Error)
	terrExpect := terrors.Timeout("cancelled", "Request already cancelled", nil)
	suite.Assert().Equal(terrExpect.Message, terr.Message)
	suite.Assert().Equal(terrExpect.Code, terr.Code)
}
