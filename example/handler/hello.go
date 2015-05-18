package handler

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/mondough/typhon/example/proto/hello"
	"github.com/mondough/typhon/server"
)

var Hello = &server.Endpoint{
	Name:     "hello",
	Handler:  helloHandler,
	Request:  &hello.Request{},
	Response: &hello.Response{},
}

// Hello is a handler that responds to a hello request with a greeting
func helloHandler(req server.Request) (proto.Message, error) {

	// Cast req.Body() (unmarshalled for you by the server) into the type you're expecting
	reqProto := req.Body().(*hello.Request)

	// Build response
	resp := &hello.Response{
		Greeting: fmt.Sprintf("Hello, %s!", reqProto.Name),
	}

	return resp, nil
}
