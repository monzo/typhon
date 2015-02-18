package main

import (
	"time"

	log "github.com/cihub/seelog"
	"github.com/golang/protobuf/proto"
	"github.com/vinceprignano/bunny/client"
	"github.com/vinceprignano/bunny/server"

	"github.com/vinceprignano/bunny/example/handler"
	"github.com/vinceprignano/bunny/example/proto/hello"
)

// main is the definition of our server
func main() {

	// Initialise our Server
	server.Initialise(&server.Config{
		Name:        "helloworld",
		Description: "Demo service which replies to a name with a greeting",
	})

	// Register an example endpoint
	server.RegisterEndpoint(&server.DefaultEndpoint{
		EndpointName: "sayhello",    // Routing Key
		Handler:      handler.Hello, // HandlerFunc
	})

	// Fire off a request to be sent back to us in 1 second
	go testHandler()

	// Start our server and serve requests
	server.Run()
}

// testHandler sends a request to our example server
func testHandler() {
	client.InitDefault("helloworld")
	time.Sleep(1 * time.Second)

	// Build and dispatch request
	req := &hello.Request{Name: proto.String("Bunny")}
	resp := &hello.Response{}
	client.Request(
		"helloworld.sayhello",
		req,
		resp,
	)

	// Log the response we receive
	log.Infof("[testHandler] received %s", resp)
}
