package main

import (
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/vinceprignano/bunny/client"
	"github.com/vinceprignano/bunny/example/foo"
	"github.com/vinceprignano/bunny/server"
)

var bunnyServer *server.Server

func HelloHandler(req *server.Request) (proto.Message, error) {
	foo := &foo.Foo{}
	proto.Unmarshal(req.Body(), foo)
	foo.Value = proto.String(fmt.Sprintf("Hello, %s!", *foo.Value))
	return foo, nil
}

func testHandler() {
	time.Sleep(1 * time.Second)
	bunnyClient := client.NewClient("test")
	bunnyClient.Init()
	req := &foo.Foo{Value: proto.String("Bunny")}
	res := &foo.Foo{}
	bunnyClient.Call("helloworld.sayhello", req, res)
	fmt.Println(res)
}

func main() {
	bunnyServer = server.NewServer("helloworld")
	bunnyServer.RegisterEndpoint(&server.DefaultEndpoint{
		EndpointName: "sayhello",
		Handler:      HelloHandler,
	})
	bunnyServer.Init()
	go testHandler()
	bunnyServer.Run()
}
