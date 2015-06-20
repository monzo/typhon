package rabbit

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/obeattie/typhon/transport"
)

func TestRabbitTransportSuite(t *testing.T) {
	suite.Run(t, new(rabbitTransportSuite))
}

type rabbitTransportSuite struct {
	transport.TransportTestSuite
}

func (suite *rabbitTransportSuite) SetupTest() {
	suite.Transport = NewTransport()
	suite.TransportTestSuite.SetupTest()
}
