package main

import (
	"fmt"
	"time"

	log "github.com/cihub/seelog"
	"github.com/golang/protobuf/proto"
	"github.com/vinceprignano/bunny"
	"github.com/vinceprignano/bunny/example/foo"
	"github.com/vinceprignano/bunny/server"
)

var service *bunny.Service

func HelloHandler(req *server.Request) (proto.Message, error) {
	foo := &foo.Foo{}
	proto.Unmarshal(req.Body(), foo)
	foo.Value = proto.String(fmt.Sprintf("Hello, %s!", *foo.Value))
	return foo, nil
}

func testHandler() {
	time.Sleep(1 * time.Second)
	req := &foo.Foo{Value: proto.String("Bunny")}
	res := &foo.Foo{}
	service.Client.Call("helloworld.sayhello", req, res)
	log.Infof("[testHandler] received %s", res)
}

func main() {
	service = bunny.NewService("helloworld")
	service.Server.RegisterEndpoint(&server.DefaultEndpoint{
		EndpointName: "sayhello",
		Handler:      HelloHandler,
	})
	go testHandler()
	service.Server.Run()
}
