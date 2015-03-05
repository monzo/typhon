// this file tests error passing between services

package test

import (
	"testing"

	"github.com/b2aio/typhon/client"
	"github.com/b2aio/typhon/errors"
	"github.com/b2aio/typhon/example/proto/callhello"
	"github.com/b2aio/typhon/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorPropagation(t *testing.T) {

	s := InitServer(t, "test")
	defer s.Close()

	var (
		errorDescription = "failure"
		errorCode        = "some.error"
	)

	// Register test endpoints
	s.RegisterEndpoint(&server.DefaultEndpoint{
		EndpointName: "callerror",
		Handler: func(req server.Request) (server.Response, error) {
			// simulate some failure
			return nil, errors.InternalService(errorCode, errorDescription)
		},

		// for convienience use example request & response
		Request:  &callhello.Request{},
		Response: &callhello.Response{},
	})

	// call the service
	resp := &callhello.Response{}
	err := client.Request(
		nil,                                // context
		"test",                             // service
		"callerror",                        // service endpoint to call
		&callhello.Request{Value: "Bunny"}, // request
		resp, // response
	)

	// Type assert this to a service error
	svcErr, ok := err.(*errors.ServiceError)
	require.Equal(t, true, ok)

	// Assert our error matches
	require.NotNil(t, err)
	assert.Equal(t, errorCode, svcErr.Code())
	assert.Equal(t, errorDescription, svcErr.Description())
	assert.Equal(t, errorDescription, svcErr.Error())

	// Also resp should be nil
	// assert.Nil(t, resp)

	// require.Equal(t, "example.hello says 'Hello, Bunny!'", resp.Value)
	// Log the response we receive
}
