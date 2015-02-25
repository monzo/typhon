package client

import (
	"time"

	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
)

type Client interface {
	Init()
	Call(ctx context.Context, serviceName, endpoint string, req proto.Message, res proto.Message) error
}

var defaultTimeout time.Duration = 1 * time.Second

var DefaultClient Client = NewRabbitClient()

func Request(ctx context.Context, serviceName, endpoint string, req proto.Message, res proto.Message) error {
	return DefaultClient.Call(ctx, serviceName, endpoint, req, res)
}
