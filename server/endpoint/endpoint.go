package endpoint

import (
	"github.com/golang/protobuf/proto"
	"github.com/streadway/amqp"
)

type Endpoint interface {
	Name() string
	HandleRequest(*amqp.Delivery) ([]byte, error)
}

type ProtoEndpoint struct {
	EndpointName string
	Handler      func(delivery *amqp.Delivery) (proto.Message, error)
}

func (p *ProtoEndpoint) Name() string {
	return p.EndpointName
}

func (p *ProtoEndpoint) HandleRequest(delivery *amqp.Delivery) ([]byte, error) {
	return p.Handler(delivery)
}

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
