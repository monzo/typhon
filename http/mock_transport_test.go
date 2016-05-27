package httpsvc

import (
	"testing"
)

func TestMockTransport(t *testing.T) {
	trans := MockTransport()
	TransportTester(trans)(t)
}
