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
	req := &hello.Request{Name: proto.String("Bunny")}
	resp := &hello.Response{}
	client.Request("helloworld.sayhello", req, resp)
	log.Infof("[testHandler] received %s", resp)
}
