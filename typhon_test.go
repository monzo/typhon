package typhon

import (
	"testing"
	"time"

	"github.com/b2aio/typhon/client"
	"github.com/b2aio/typhon/example/proto/callhello"
	"github.com/b2aio/typhon/server"
	"github.com/stretchr/testify/require"

	"github.com/b2aio/typhon/example/handler"
)

func initServer(t *testing.T) {
	// Initialize our Server
	server.Init(&server.Config{
		Name:        "example",
		Description: "Example service",
	})

	// Register an example endpoint
	server.RegisterEndpoint(handler.Hello)
	server.RegisterEndpoint(handler.CallHello)

	go server.Run()

	select {
	case <-server.NotifyConnected():
	case <-time.After(1 * time.Second):
		t.Fatalf("StubServer couldn't connect to RabbitMQ")
	}
}

func TestExample(t *testing.T) {

	initServer(t)

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
