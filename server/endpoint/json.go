package endpoint

import "github.com/streadway/amqp"

type JsonEndpoint struct {
	EndpointName string
	Handler      func(delivery *amqp.Delivery) (interface{}, error)
}

func (j *JsonEndpoint) Name() string {
	return j.EndpointName
}

func (j *JsonEndpoint) HandleRequest(delivery *amqp.Delivery) ([]byte, error) {
	return j.Handler(delivery)
}
