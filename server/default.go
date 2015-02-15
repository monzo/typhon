package server

import "github.com/golang/protobuf/proto"

var defaultServer Server

type defaultHandler func(req *Request) (proto.Message, error)

var InitDefault = func(name string) {
	defaultServer = NewRabbitServer(name)
	defaultServer.Init()
}

func RegisterDefaultEndpoint(name string, handler defaultHandler) {
	defaultServer.RegisterEndpoint(&DefaultEndpoint{
		EndpointName: name,
		Handler:      handler,
	})
}

func Run() {
	defaultServer.Run()
}
