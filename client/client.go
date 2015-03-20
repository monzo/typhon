package client

import (
	"time"

	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
)

type Client interface {
	Init()
	Req(ctx context.Context, service, endpoint string, req proto.Message, res proto.Message) error
	CustomReq(req *Request) (*Response, error)
}

var defaultTimeout time.Duration = 1 * time.Second

var DefaultClient Client = NewRabbitClient()

// Req sends a request to a service using the DefaultClient
// and unmarshals the response into the supplied protobuf
func Req(ctx context.Context, service, endpoint string, req proto.Message, res proto.Message) error {
	return DefaultClient.Req(ctx, service, endpoint, req, res)
}

// CustomReq sends a raw request using the DefaultClient
// without the usual marshaling helpers
func CustomReq(req *Request) (*Response, error) {
	return DefaultClient.CustomReq(req)
}
