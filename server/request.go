package server

import "golang.org/x/net/context"

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
}
