package main

import (
	"github.com/vinceprignano/bunny/server"
	"github.com/vinceprignano/bunny/transport/rabbit"
)

func main() {
	server.NewServer("hello", rabbit.NewRabbitTransport())
}
