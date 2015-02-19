package handler

import (
	"fmt"

	"github.com/golang/protobuf/proto"

	"github.com/b2aio/typhon/example/proto/hello"
	"github.com/b2aio/typhon/server"
)

// Hello is a handler that responds to a hello request with a greeting
func Hello(req server.Request) (server.Response, error) {

	// Unmarshal our request
	f := &hello.Request{}
	if err := proto.Unmarshal(req.Body(), f); err != nil {
		return nil, fmt.Errorf("Count not unmarshal request")
	}

	// Get a value from our unmarshalled protobuf
	name := f.GetName()

	// Do something here

	// Build response
	resp := server.NewProtoResponse(&hello.Response{
		Greeting: proto.String(fmt.Sprintf("Hello, %s!", name)),
	})

	return resp, nil
}
