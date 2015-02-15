package server

import (
	"github.com/golang/protobuf/proto"
	"github.com/vinceprignano/bunny/rabbit"
)

var defaultServer *RabbitServer

type defaultHandler func(req *Request) (proto.Message, error)

var InitDefault = func(name string) {
	defaultServer = &RabbitServer{
		ServiceName:      name,
		endpointRegistry: NewEndpointRegistry(),
		connection:       rabbit.NewRabbitConnection(),
	}
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
