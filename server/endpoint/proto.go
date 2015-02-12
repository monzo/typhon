package endpoint

import (
	"github.com/golang/protobuf/proto"
	"github.com/streadway/amqp"
)

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
