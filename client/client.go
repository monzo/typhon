package client

import "github.com/golang/protobuf/proto"

type Client interface {
	Init()
	Call(serviceName, endpoint string, req proto.Message, res proto.Message) error
}
