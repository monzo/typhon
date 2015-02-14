package server

import "github.com/streadway/amqp"

type Request struct {
	delivery *amqp.Delivery
}

func NewRequest(delivery *amqp.Delivery) *Request {
	return &Request{
		delivery: delivery,
	}
}

func (r *Request) Body() []byte {
	return r.delivery.Body
}

func (r *Request) CorrelationID() string {
	return r.delivery.CorrelationId
}

func (r *Request) ReplyTo() string {
	return r.delivery.ReplyTo
}

func (r *Request) RoutingKey() string {
	return r.delivery.RoutingKey
}

func (r *Request) Interface() interface{} {
	return r.delivery
}
