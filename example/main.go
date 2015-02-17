package main

import (
	"time"

	log "github.com/cihub/seelog"
	"github.com/golang/protobuf/proto"
	"github.com/vinceprignano/bunny/client"
	"github.com/vinceprignano/bunny/server"

	"github.com/vinceprignano/bunny/example/foo"
	"github.com/vinceprignano/bunny/example/handler"
)

// main is the definition of our server
func main() {

	// Initialise our Server
	server.DefaultServer = server.NewRabbitServer("helloworld")

	// Register and endpoint
	server.RegisterEndpoint("sayhello", handler.HelloHandler)

	// Fire off a request to be sent back to us in 1 second
	go testHandler()

	// Start our server and serve requests
	server.Run()
}

// testHandler sends a request to our example server
func testHandler() {
	client.InitDefault("helloworld")
	time.Sleep(1 * time.Second)
	req := &foo.Foo{Value: proto.String("Bunny")}
	res := &foo.Foo{}
	client.Request("helloworld.sayhello", req, res)
	log.Infof("[testHandler] received %s", res)
}
