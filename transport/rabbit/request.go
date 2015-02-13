package rabbit

import "github.com/streadway/amqp"

type RabbitRequest struct {
	delivery *amqp.Delivery
}

func NewRabbitRequest(delivery *amqp.Delivery) *RabbitRequest {
	return &RabbitRequest{
		delivery: delivery,
	}
}

func (r *RabbitRequest) Body() []byte {
	return r.delivery.Body
}

func (r *RabbitRequest) CorrelationID() string {
	return r.delivery.CorrelationId
}

func (r *RabbitRequest) ReplyTo() string {
	return r.delivery.ReplyTo
}

func (r *RabbitRequest) RoutingKey() string {
	return r.delivery.RoutingKey
}

func (r *RabbitRequest) Interface() interface{} {
	return r.delivery
}
