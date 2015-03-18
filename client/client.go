package client

import (
	"time"

	"github.com/b2aio/typhon/errors"
	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
)

type Client interface {
	Init()
	Call(ctx context.Context, service, endpoint string, req proto.Message, res proto.Message) *errors.Error
}

var defaultTimeout time.Duration = 1 * time.Second

var DefaultClient Client = NewRabbitClient()

// Request sends a request to a service using the DefaultClient
func Request(ctx context.Context, service, endpoint string, req proto.Message, res proto.Message) *errors.Error {
	return DefaultClient.Call(ctx, service, endpoint, req, res)
}
