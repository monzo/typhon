package server

import (
	"github.com/mondough/typhon/auth"
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

	// Session provided on this request. Recovers session on first call.
	Session() (auth.Session, error)
	// SetSession for this request, useful at api level and for mocking
	SetSession(auth.Session)

	// HasRecoveredSession returns true if the session was previously successfully
	// recovered from the access token
	// @todo this is ugly and needs to be refactored
	HasRecoveredSession() bool

	AccessToken() string
	TraceID() string
	ParentRequestID() string

	// Server is a reference to the server currently processing this request
	Server() Server
}

// RecoverServerFromContext retrieves the request in which this context is executing
// This is used when making nested requests
func RecoverRequestFromContext(ctx context.Context) Request {
	if req, ok := ctx.(Request); ok {
		return req
	}
	return nil
}
