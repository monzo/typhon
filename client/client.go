package client

import (
	"time"

	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
)

type Client interface {
	Init()
	Request(ctx context.Context, service, endpoint string, req proto.Message, res proto.Message) error
}

var defaultTimeout time.Duration = 1 * time.Second

var DefaultClient Client = NewRabbitClient()

// Request sends a request to a service using the DefaultClient
func Request(ctx context.Context, service, endpoint string, req proto.Message, res proto.Message) error {
	return DefaultClient.Request(ctx, service, endpoint, req, res)
}
