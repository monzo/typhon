package handler

import (
	"fmt"

	"github.com/mondough/typhon/client"
	"github.com/mondough/typhon/example/proto/callhello"
	"github.com/mondough/typhon/example/proto/hello"
	"github.com/mondough/typhon/server"
	"github.com/golang/protobuf/proto"
)

var CallHello = &server.Endpoint{
	Name:     "callhello",
	Handler:  callhelloHandler,
	Request:  &callhello.Request{},
	Response: &callhello.Response{},
}

func callhelloHandler(req server.Request) (proto.Message, error) {

	reqProto := req.Body().(*callhello.Request)
	resp := &hello.Response{}

	client.Req(
		req,
		"example",
		"hello",
		&hello.Request{
			Name: reqProto.Value,
		},
		resp,
	)

	return &callhello.Response{
		Value: fmt.Sprintf("example.hello says '%s'", resp.Greeting),
	}, nil
}
