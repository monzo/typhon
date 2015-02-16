package server

import "github.com/golang/protobuf/proto"

var DefaultServer Server

type defaultHandler func(req Request) (proto.Message, error)

var InitDefault = func(name string) {
	DefaultServer = NewRabbitServer(name)
	DefaultServer.Init()
}

func RegisterDefaultEndpoint(name string, handler defaultHandler) {
	DefaultServer.RegisterEndpoint(&DefaultEndpoint{
		EndpointName: name,
		Handler:      handler,
	})
}

func Run() {
	DefaultServer.Run()
}
