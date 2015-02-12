package handler

import (
	"github.com/golang/protobuf/proto"
	"github.com/streadway/amqp"
)

type ProtoHandler struct{}

func (p *ProtoHandler) HandleDelivery(delivery amqp.Delivery) {

}

func (p *ProtoHandler) Marshal(v interface{}) ([]byte, error) {
	return proto.Marshal(v.(proto.Message))
}

func (p *ProtoHandler) Unmarshal(data []byte, v interface{}) error {
	return proto.Unmarshal(data, v.(proto.Message))
}
