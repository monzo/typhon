package server

import (
	"strings"

	"github.com/b2aio/typhon/client"
	"github.com/golang/protobuf/proto"
	"github.com/streadway/amqp"
	"golang.org/x/net/context"
)

type AMQPRequest struct {
	context.Context
	body     interface{}
	delivery *amqp.Delivery
}

func NewAMQPRequest(delivery *amqp.Delivery) *AMQPRequest {
	return &AMQPRequest{
		delivery: delivery,
	}
}

// RabbitMQ / AMQP fields

func (r *AMQPRequest) Service() string {
	routingKey := r.RoutingKey()
	lastDotIndex := strings.LastIndex(routingKey, ".")
	if lastDotIndex == -1 {
		return routingKey
	}
	return routingKey[:lastDotIndex]
}

func (r *AMQPRequest) Endpoint() string {
	routingKey := r.RoutingKey()
	lastDotIndex := strings.LastIndex(routingKey, ".")
	if lastDotIndex == -1 {
		return ""
	}
	return routingKey[lastDotIndex+1:]
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
	return r.delivery.CorrelationId
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

// Client implementation

// ScopedRequest calls a service within the scope of the current request
// This allows request context to be passed transparently,
// and child requests to be 'scoped' within a parent request
func (r *AMQPRequest) ScopedRequest(service string, endpoint string, req proto.Message, resp proto.Message) error {
	// Temporarily just call the default client
	// This means we can nail down our external interface, and work the internals out properly
	return client.Request(r, service, endpoint, req, resp)
}
