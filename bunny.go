package bunny

import (
	"github.com/vinceprignano/bunny/server"
	"github.com/vinceprignano/bunny/transport"
	"github.com/vinceprignano/bunny/transport/rabbit"
)

func NewServer(name string, tp transport.Transport) *server.Server {
	return server.NewServer(name, tp)
}

func NewRabbitServer(name string) *server.Server {
	return server.NewServer(name, rabbit.NewRabbitTransport())
}
