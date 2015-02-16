package server

import "github.com/streadway/amqp"

type Request interface {
	Body() []byte
}

type AMQPRequest struct {
	delivery *amqp.Delivery
}

func NewAMQPRequest(delivery *amqp.Delivery) *AMQPRequest {
	return &AMQPRequest{
		delivery: delivery,
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
