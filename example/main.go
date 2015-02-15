package main

import (
	"fmt"
	"time"

	log "github.com/cihub/seelog"
	"github.com/golang/protobuf/proto"
	"github.com/vinceprignano/bunny/client"
	"github.com/vinceprignano/bunny/example/foo"
	"github.com/vinceprignano/bunny/server"
)

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
	client.Request("helloworld.sayhello", req, res)
	log.Infof("[testHandler] received %s", res)
}

func main() {
	client.InitDefault("helloworld")
	server.InitDefault("helloworld")
	server.RegisterDefaultEndpoint("sayhello", HelloHandler)
	go testHandler()
	server.Run()
}
