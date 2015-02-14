package server

import "github.com/golang/protobuf/proto"

type Endpoint interface {
	Name() string
	HandleRequest(req *Request) (proto.Message, error)
}

type DefaultEndpoint struct {
	EndpointName string
	Handler      func(req *Request) (proto.Message, error)
}

func (d *DefaultEndpoint) Name() string {
	return d.EndpointName
}

func (d *DefaultEndpoint) HandleRequest(req *Request) (proto.Message, error) {
	return d.Handler(req)
}
