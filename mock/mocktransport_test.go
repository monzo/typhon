package mock

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/monzo/typhon/transport"
)

func TestMockTransportSuite(t *testing.T) {
	suite.Run(t, new(mockTransportSuite))
}

type mockTransportSuite struct {
	transport.TransportTestSuite
}

func (suite *mockTransportSuite) SetupTest() {
	suite.Transport = NewTransport()
	suite.TransportTestSuite.SetupTest()
}
