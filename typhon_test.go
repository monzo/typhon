package typhon

import (
	"testing"
	"time"

	"github.com/b2aio/typhon/client"
	"github.com/b2aio/typhon/example/proto/callhello"
	"github.com/b2aio/typhon/server"
	"github.com/golang/protobuf/proto"
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
	case <-server.DefaultServer.NotifyConnected():
	case <-time.After(1 * time.Second):
		t.Fatalf("StubServer couldn't connect to RabbitMQ")
	}
}

func TestExample(t *testing.T) {

	initServer(t)

	client.InitDefault("helloworld")

	resp := &callhello.Response{}
	client.Request(
		"example.callhello",
		&callhello.Request{Value: proto.String("Bunny")},
		resp,
	)

	require.Equal(t, "example.hello says 'Hello, Bunny!'", resp.GetValue())
	// Log the response we receive
	t.Logf("[testHandler] received %s", resp)
}
