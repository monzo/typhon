package client

import "github.com/golang/protobuf/proto"

var DefaultClient Client

var InitDefault = func(name string) {
	DefaultClient = NewRabbitClient(name)
	DefaultClient.Init()
}

func Request(routingKey string, req proto.Message, res proto.Message) error {
	return DefaultClient.Call(routingKey, req, res)
}
