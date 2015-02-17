package handler

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/vinceprignano/bunny/example/proto/hello"
	"github.com/vinceprignano/bunny/server"
)

func HelloHandler(req server.Request) (proto.Message, error) {

	f := &hello.Request{}
	if err := proto.Unmarshal(req.Body(), f); err != nil {
		return nil, fmt.Errorf("Count not unmarshal request")
	}

	// Get a value from our unmarshalled protobuf
	name := f.GetName()

	// Do something here

	// Build response
	rsp := &hello.Response{
		Greeting: proto.String(fmt.Sprintf("Hello, %s!", name)),
	}

	return rsp, nil
}
