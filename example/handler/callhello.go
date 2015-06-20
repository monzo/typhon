package handler

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/obeattie/typhon/client"
	"github.com/obeattie/typhon/example/proto/callhello"
	"github.com/obeattie/typhon/example/proto/hello"
	"github.com/obeattie/typhon/server"
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
