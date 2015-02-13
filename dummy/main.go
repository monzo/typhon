package main

import (
	"github.com/vinceprignano/bunny/server"
	"github.com/vinceprignano/bunny/transport/rabbit"
)

func main() {
	bunnyServer := server.NewServer("hello", rabbit.NewRabbitTransport())
	bunnyServer.Init()
}
