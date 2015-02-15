package client

import "github.com/golang/protobuf/proto"

var defaultClient Client

var InitDefault = func(name string) {
	defaultClient = NewRabbitClient(name)
	defaultClient.Init()
}

func Request(routingKey string, req proto.Message, res proto.Message) error {
	return defaultClient.Call(routingKey, req, res)
}
