package main

import (
	"github.com/vinceprignano/bunny/server"
	"github.com/vinceprignano/bunny/transport"
)

func main() {
	bunnyServer := server.NewServer("hello", transport.NewRabbitTransport())
	bunnyServer.Init()
}
