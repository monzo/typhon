package main

import (
	"github.com/golang/protobuf/proto"
	"github.com/vinceprignano/bunny/dummy/foo"
	"github.com/vinceprignano/bunny/server"
)

var bunnyServer *server.Server

func HelloHandler(req *server.Request) (proto.Message, error) {
	foo := &foo.Foo{}
	proto.Unmarshal(req.Body(), foo)
	return foo, nil
}

func main() {
	bunnyServer = server.NewServer("helloworld")
	bunnyServer.RegisterEndpoint(&server.DefaultEndpoint{
		EndpointName: "sayhello",
		Handler:      HelloHandler,
	})
	bunnyServer.Init()
	bunnyServer.Run()
}
