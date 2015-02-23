package client

import "github.com/golang/protobuf/proto"

type Client interface {
	Init()
	Call(serviceName, endpoint string, req proto.Message, res proto.Message) error
}

var DefaultClient Client = NewRabbitClient()

func Request(serviceName, endpoint string, req proto.Message, res proto.Message) error {
	return DefaultClient.Call(serviceName, endpoint, req, res)
}
