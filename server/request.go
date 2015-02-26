package server

import (
	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
)

type Request interface {
	context.Context

	// Payload are the actual bytes returned from the transport
	Payload() []byte

	// Body is the Unmarshalled `Payload()`. If `RequestType()` is set on
	// the `Endpoint`, we can attempt to unmarshal it for you
	Body() interface{}
	SetBody(interface{})

	// Details about the service and endpoint being called
	Service() string
	Endpoint() string

	// ScopedRequest makes a client request within the scope of the current request
	// @todo change the request & response interface to decouple from protobuf
	ScopedRequest(service string, endpoint string, req proto.Message, resp proto.Message) error
}
