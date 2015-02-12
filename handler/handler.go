package handler

import "github.com/streadway/amqp"

type Handler interface {
	HandleDelivery(amqp.Delivery)
	Marshal(interface{}) ([]byte, error)
	Unmarshal([]byte, interface{})
}
