package server

import (
	"strings"

	"github.com/streadway/amqp"
	"golang.org/x/net/context"
)

type Request interface {
	context.Context

	Body() []byte
	ServiceName() string
	Endpoint() string
}

type AMQPRequest struct {
	context.Context

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
	} else {
		return routingKey[:lastDotIndex]
	}
}

func (r *AMQPRequest) Endpoint() string {
	routingKey := r.RoutingKey()
	lastDotIndex := strings.LastIndex(routingKey, ".")
	if lastDotIndex == -1 {
		return ""
	} else {
		return routingKey[lastDotIndex+1:]
	}
}

func (r *AMQPRequest) Body() []byte {
	return r.delivery.Body
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
