package server

import (
	"github.com/b2aio/typhon/auth"
	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
)

// Request received by the server
type Request interface {
	context.Context

	// Id of this message, used to correlate the response
	Id() string
	// ContentType of the payload
	ContentType() string
	// Payload of raw bytes received from the transport
	Payload() []byte
	// Body is the Unmarshalled `Payload()`. If `RequestType()` is set on
	// the `Endpoint`, we can attempt to unmarshal it for you
	Body() interface{}
	// SetBody of this request
	SetBody(interface{})
	// Service which this request was intended for
	Service() string
	// Endpoint to be called on the receiving service
	Endpoint() string
	// ScopedRequest makes a client request within the scope of the current request
	// @todo change the request & response interface to decouple from protobuf
	ScopedRequest(service string, endpoint string, req proto.Message, resp proto.Message) error

	// Session provided on this request
	Session() auth.Session
	// SetCredentials for this request, useful at api level and for mocking
	SetSession() auth.Session
}
