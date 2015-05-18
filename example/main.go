package main

import (
	"github.com/mondough/typhon/server"

	"github.com/mondough/typhon/example/handler"
)

// main is the definition of our server
func main() {

	// Initialize our Server
	server.Init(&server.Config{
		Name:        "example",
		Description: "Example service",
	})

	// Register an example endpoint
	server.RegisterEndpoint(handler.Hello)
	server.RegisterEndpoint(handler.CallHello)

	// Start our server and serve requests
	server.Run()
}
