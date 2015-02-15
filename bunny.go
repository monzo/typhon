package bunny

import (
	"github.com/vinceprignano/bunny/client"
	"github.com/vinceprignano/bunny/server"
)

type Service struct {
	Client client.Client
	Server server.Server
}

var NewService = func(name string) *Service {
	cl := client.NewRabbitClient(name)
	cl.Init()
	srv := server.NewRabbitServer(name)
	srv.Init()
	return &Service{Client: cl, Server: srv}
}
