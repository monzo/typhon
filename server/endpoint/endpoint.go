package endpoint

import "github.com/streadway/amqp"

type Endpoint interface {
	Name() string
	HandleRequest(*amqp.Delivery) ([]byte, error)
}
