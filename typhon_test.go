package typhon

import (
	"testing"

	"github.com/b2aio/typhon/client"
	"github.com/b2aio/typhon/example/handler"
	"github.com/b2aio/typhon/example/proto/callhello"
	"github.com/b2aio/typhon/test"
	"github.com/stretchr/testify/require"
)

func TestExample(t *testing.T) {

	s := test.InitServer(t, "example")
	defer s.Close()

	// Register example endpoints
	s.RegisterEndpoint(handler.Hello)
	s.RegisterEndpoint(handler.CallHello)

	resp := &callhello.Response{}
	client.Request(
		nil,                                // context
		"example",                          // service
		"callhello",                        // service endpoint to call
		&callhello.Request{Value: "Bunny"}, // request
		resp, // response
	)
	t.Logf("[testHandler] received %s", resp)
	require.Equal(t, "example.hello says 'Hello, Bunny!'", resp.Value)
	// Log the response we receive
}
