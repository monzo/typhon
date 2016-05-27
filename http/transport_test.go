package httpsvc

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
)

func TransportTester(trans Transport) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()
		suite.Run(t, &transportTestSuite{
			trans: trans})
	}
}

func TestNetworkTransport(t *testing.T) {
	trans := NetworkTransport("127.0.0.1:30001")
	TransportTester(trans)(t)
}

type transportTestSuite struct {
	suite.Suite
	trans Transport
}

func (suite *transportTestSuite) TearDownSuite() {
	suite.trans.Close(0)
}

func (suite *transportTestSuite) TestStraightforward() {
	trans := suite.trans
	svc := Service(func(req Request) Response {
		return NewResponse(http.StatusOK, nil)
	})
	trans.Listen("service.test", svc)

	req := NewRequest(nil, "GET", "http://127.0.0.1:30001")
	req.Host = "service.test"
	rsp := trans.Send(req)
	suite.Require().NotNil(rsp)
	suite.Assert().NoError(rsp.Error)
	suite.Assert().Equal(rsp.Response.StatusCode, http.StatusOK)
}
