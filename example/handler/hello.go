package handler

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/vinceprignano/bunny/example/foo"
	"github.com/vinceprignano/bunny/server"
)

func HelloHandler(req server.Request) (proto.Message, error) {
	foo := &foo.Foo{}
	proto.Unmarshal(req.Body(), foo)
	foo.Value = proto.String(fmt.Sprintf("Hello, %s!", *foo.Value))
	return foo, nil
}
