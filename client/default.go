package client

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/nu7hatch/gouuid"
	"github.com/vinceprignano/bunny/rabbit"
)

var defaultClient *RabbitClient

var InitDefault = func(name string) {
	uuidQueue, _ := uuid.NewV4()
	defaultClient = &RabbitClient{
		Name:       name,
		dispatcher: newDispatcher(),
		connection: rabbit.NewRabbitConnection(),
		replyTo:    fmt.Sprintf("replyTo-%s-%s", name, uuidQueue.String()),
	}
	defaultClient.Init()
}

func Request(routingKey string, req proto.Message, res proto.Message) error {
	return defaultClient.Call(routingKey, req, res)
}
