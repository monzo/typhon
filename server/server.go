package server

import "github.com/golang/protobuf/proto"

type Server interface {
	Init()
	Run()

	RegisterEndpoint(endpoint Endpoint)
	DeregisterEndpoint(pattern string)
}

var DefaultServer Server

type defaultHandler func(req Request) (proto.Message, error)

func RegisterEndpoint(name string, handler defaultHandler) {
	DefaultServer.RegisterEndpoint(&DefaultEndpoint{
		EndpointName: name,
		Handler:      handler,
	})
}

func Run() {
	DefaultServer.Run()
}
