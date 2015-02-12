package endpoint

import (
	"encoding/json"

	"github.com/golang/protobuf/proto"
	"github.com/streadway/amqp"
)

type Endpoint interface {
	Name() string
	HandleRequest(*amqp.Delivery) ([]byte, error)
}

type ProtoEndpoint struct {
	name    string
	Handler func(delivery *amqp.Delivery) (proto.Message, error)
}

func (p *ProtoEndpoint) Name() string {
	return p.name
}

func (p *ProtoEndpoint) HandleRequest(delivery *amqp.Delivery) ([]byte, error) {
	return p.Handler(delivery)
}

type JsonEndpoint struct {
	name    string
	Handler func(delivery map[string]interface{}) (map[string]interface{}, error)
}

func (j *JsonEndpoint) Name() string {
	return j.name
}

func (j *JsonEndpoint) HandleRequest(delivery *amqp.Delivery) ([]byte, error) {
	v := make(map[string]interface{})
	json.Unmarshal(delivery.Body, v)
	return j.Handler(v)
}
