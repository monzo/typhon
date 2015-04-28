package server

import (
	"github.com/golang/protobuf/proto"
	"github.com/streadway/amqp"
	"golang.org/x/net/context"

	"github.com/b2aio/typhon/client"
)

type AMQPRequest struct {
	context.Context
	body     interface{}
	delivery *amqp.Delivery

	id              string
	contentType     string
	contentEncoding string
	service         string
	endpoint        string

	accessToken string
}

func NewAMQPRequest(delivery *amqp.Delivery) *AMQPRequest {

	contentType, _ := delivery.Headers["Content-Type"].(string)
	contentEncoding, _ := delivery.Headers["Content-Encoding"].(string)
	service, _ := delivery.Headers["Service"].(string)
	endpoint, _ := delivery.Headers["Endpoint"].(string)
	accessToken, _ := delivery.Headers["Access-Token"].(string)

	return &AMQPRequest{
		delivery: delivery,

		id:              delivery.CorrelationId,
		contentType:     contentType,
		contentEncoding: contentEncoding,
		service:         service,
		endpoint:        endpoint,
		accessToken:     accessToken,
	}
}

// RabbitMQ / AMQP fields

func (r *AMQPRequest) Id() string {
	return r.id
}

func (r *AMQPRequest) ContentType() string {
	return r.contentType
}

func (r *AMQPRequest) ContentEncoding() string {
	return r.contentEncoding
}

func (r *AMQPRequest) Service() string {
	return r.service
}

func (r *AMQPRequest) Endpoint() string {
	return r.endpoint
}

func (r *AMQPRequest) Payload() []byte {
	return r.delivery.Body
}

func (r *AMQPRequest) Body() interface{} {
	return r.body
}

func (r *AMQPRequest) SetBody(body interface{}) {
	r.body = body
}

func (r *AMQPRequest) CorrelationID() string {
	return r.id
}

func (r *AMQPRequest) ReplyTo() string {
	return r.delivery.ReplyTo
}

func (r *AMQPRequest) RoutingKey() string {
	return r.delivery.RoutingKey
}

func (r *AMQPRequest) Interface() interface{} {
	return r.delivery
}

func (r *AMQPRequest) AccessToken() string {
	return r.accessToken
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
