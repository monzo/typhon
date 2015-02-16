package server

import (
	"github.com/streadway/amqp"
	"golang.org/x/net/context"
)

type Request interface {
	context.Context

	Body() []byte
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
