package rabbit

import "github.com/streadway/amqp"

type RabbitRequest struct {
	delivery *amqp.Delivery
}

func NewRabbitRequest() *RabbitRequest {
	return &Request{
		delivery: delivery,
	}
}

func (r *RabbitRequest) Body() []byte {
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
