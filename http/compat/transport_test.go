package httpcompat

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/mondough/typhon/http"
	"github.com/mondough/typhon/transport"
)

func TestNew2OldTransport(t *testing.T) {
	trans := httpsvc.MockTransport()
	defer trans.Close(0)
	suite.Run(t, new(new2OldTransportSuite))
}

type new2OldTransportSuite struct {
	transport.TransportTestSuite
}

func (suite *new2OldTransportSuite) SetupTest() {
	suite.Transport = New2OldTransport(httpsvc.MockTransport())
	suite.TransportTestSuite.SetupTest()
}
