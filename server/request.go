package server

import (
	"strings"

	"github.com/streadway/amqp"
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

	ServiceName() string
	Endpoint() string
}

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

func (r *AMQPRequest) ServiceName() string {
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
