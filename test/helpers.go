package test

import (
	"testing"
	"time"

	"github.com/b2aio/typhon/server"
	"github.com/golang/protobuf/proto"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/require"
)

var stubServer *StubServer

func InitStubServer(t *testing.T, serviceName string) *StubServer {
	if stubServer == nil {
		stubServer = NewStubServer(t, serviceName)
	}
	return stubServer
}

// Helper method to call a handler function (the function being tested)
// directly with a `proto.Message`.
// Returns errors that were returned from the handler function directly.
// Marshalling errors cause the test to fail instantly
func CallEndpoint(t *testing.T, endpoint *server.Endpoint, reqProto proto.Message, respProto proto.Message) error {
	// Call handler with amqp delivery
	reqBytes, err := proto.Marshal(reqProto)
	require.NoError(t, err)
	resp, err := endpoint.HandleRequest(server.NewAMQPRequest(&amqp.Delivery{
		// todo - add other params here
		Timestamp: time.Now().UTC(),
		Body:      reqBytes,
		Headers: amqp.Table{
			"Content-Type":     "application/x-protobuf",
			"Content-Encoding": "request",
		},
	}))
	if err != nil {
		return err
	}
	respBytes, err := proto.Marshal(resp)
	require.NoError(t, err)
	err = proto.Unmarshal(respBytes, respProto)
	require.NoError(t, err)
	return nil
}
