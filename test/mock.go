package test

import (
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/mock"
	"github.com/vinceprignano/bunny/server"
)

type BunnyTestServer struct {
	mock.Mock
	server.Server
	registry *server.Registry
}

func NewBunnyTestServer(name string) *BunnyTestServer {
	srv := &BunnyTestServer{
		registry: server.NewRegistry(),
	}
	srv.Name = name
	return srv
}

func (b *BunnyTestServer) Init() {
	b.Called()
}

func (b *BunnyTestServer) RegisterEndpoint(endpoint server.Endpoint) {
	b.registry.RegisterEndpoint(endpoint)
	b.Called(endpoint)
}

func (b *BunnyTestServer) Run() {
	b.Called()
}

type BunnyTestClient struct {
	mock.Mock
	Name           string
	resultMessages map[string]proto.Message
}

func NewBunnyTestClient(name string) *BunnyTestClient {
	return &BunnyTestClient{
		Name:           name,
		resultMessages: make(map[string]proto.Message),
	}
}

func (b *BunnyTestClient) Init() {
	b.Called()
}

func (b *BunnyTestClient) Call(routingKey string, req proto.Message, res proto.Message) error {
	b.Called(routingKey, req, res)
	return nil
}
