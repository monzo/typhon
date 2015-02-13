package rabbit

type RabbitRequest struct {
	delivery *ampq.Delivery
}

func NewRabbitRequest() *RabbitRequest {
	return &Request{
		delivery: delivery,
	}
}

func (r *RabbitRequest) Body() []byte {
	return r.delivery.Body
}
