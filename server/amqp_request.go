package server

import (
	"github.com/golang/protobuf/proto"
	"github.com/streadway/amqp"
	"golang.org/x/net/context"

	"github.com/b2aio/typhon/auth"
	"github.com/b2aio/typhon/client"
)

// AMQPRequest satisfies our server Request interface, and
// represents requests delivered via an AMQP implementation
type AMQPRequest struct {
	context.Context

	// reference to the server handling this request
	s Server

	body     interface{}
	delivery *amqp.Delivery

	// session holds our authentication information
	session auth.Session

	id              string
	contentType     string
	contentEncoding string
	service         string
	endpoint        string
}

// NewAMQPRequest marshals a raw AMQP delivery to a Request interface
func NewAMQPRequest(s Server, delivery *amqp.Delivery) (*AMQPRequest, error) {
	var (
		session auth.Session
		err     error
	)

	// Handle basic headers
	contentType, _ := delivery.Headers["Content-Type"].(string)
	contentEncoding, _ := delivery.Headers["Content-Encoding"].(string)
	service, _ := delivery.Headers["Service"].(string)
	endpoint, _ := delivery.Headers["Endpoint"].(string)

	// Attempt to unmarshal the session information
	sess, _ := delivery.Headers["Session"].(string)
	if sess != "" {
		session, err = s.AuthenticationProvider().UnmarshalSession([]byte(sess))
	}
	if err != nil {
		return nil, err
	}

	return &AMQPRequest{
		s:        s,
		delivery: delivery,
		session:  session,

		id:              delivery.CorrelationId,
		contentType:     contentType,
		contentEncoding: contentEncoding,
		service:         service,
		endpoint:        endpoint,
	}, nil
}

// RabbitMQ / AMQP fields

// Id specifies the request's unique identifier
func (r *AMQPRequest) Id() string {
	return r.id
}

// ContentType of this request - eg. json, protobuf
func (r *AMQPRequest) ContentType() string {
	return r.contentType
}

// ContentEncoding of this request - eg. request, response, error
func (r *AMQPRequest) ContentEncoding() string {
	return r.contentEncoding
}

// Service which should process this request
func (r *AMQPRequest) Service() string {
	return r.service
}

// Endpoint which should process this request
func (r *AMQPRequest) Endpoint() string {
	return r.endpoint
}

// Payload is the raw delivery body
func (r *AMQPRequest) Payload() []byte {
	return r.delivery.Body
}

// Body of the request is populated if the payload was successfully
// unmarshalled to a protobuf message by our helper methods
func (r *AMQPRequest) Body() interface{} {
	return r.body
}

// SetBody of the request to a decoded payload
func (r *AMQPRequest) SetBody(body interface{}) {
	r.body = body
}

// CorrelationID allows our services to correlate requests and responses
func (r *AMQPRequest) CorrelationID() string {
	return r.id
}

// ReplyTo specifies the client to reply to
func (r *AMQPRequest) ReplyTo() string {
	return r.delivery.ReplyTo
}

// RoutingKey of the request used to route the message to this service
func (r *AMQPRequest) RoutingKey() string {
	return r.delivery.RoutingKey
}

// Interface returns the raw AMQP delivery
func (r *AMQPRequest) Interface() interface{} {
	return r.delivery
}

// Session returns the session associated with this request
func (r *AMQPRequest) Session() string {
	return r.session
}

// Client implementation

// ScopedRequest calls a service within the scope of the current request
// This allows request context to be passed transparently,
// and child requests to be 'scoped' within a parent request
func (r *AMQPRequest) ScopedRequest(service string, endpoint string, req proto.Message, resp proto.Message) error {
	// Temporarily just call the default client
	// This means we can nail down our external interface, and work the internals out properly
	// where we can initialise a 'client' and separate out the connected transport layer
	// a client in this case would allow multiple parallel requests etc.
	return client.Req(r, service, endpoint, req, resp)
}
