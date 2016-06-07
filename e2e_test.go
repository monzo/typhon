package typhon

import (
	"net/http"
	"os"
	"testing"

	"github.com/mondough/terrors"
	"github.com/stretchr/testify/suite"
)

func TestE2E(t *testing.T) {
	os.Setenv("LISTEN_ADDR", "localhost:30001")
	defer os.Unsetenv("LISTEN_ADDR")
	suite.Run(t, &e2eSuite{})
}

type e2eSuite struct {
	suite.Suite
}

func (suite *e2eSuite) TestStraightforward() {
	l := Listen(func(req Request) Response {
		return NewResponse(req)
	})
	defer l.Stop()

	req := NewRequest(nil, "GET", "http://localhost:30001", nil)
	rsp := req.Send().Response()
	suite.Assert().NoError(rsp.Error)
	suite.Assert().Equal(http.StatusOK, rsp.StatusCode)
}

func (suite *e2eSuite) TestError() {
	l := Listen(func(req Request) Response {
		return Response{
			Error: terrors.Unauthorized("ah_ah_ah", "You didn't say the magic word!", map[string]string{
				"param": "value"})}
	})
	defer l.Stop()

	req := NewRequest(nil, "GET", "http://localhost:30001", nil)
	rsp := req.Send().Response()
	suite.Assert().Equal(http.StatusUnauthorized, rsp.StatusCode)
	suite.Assert().Error(rsp.Error)
	terr := terrors.Wrap(rsp.Error, nil).(*terrors.Error)
	terrExpect := terrors.Unauthorized("ah_ah_ah", "You didn't say the magic word!", nil)
	suite.Assert().Equal(terrExpect.Message, terr.Message)
	suite.Assert().Equal(terrExpect.Code, terr.Code)
	suite.Assert().Equal("value", terr.Params["param"])
}
