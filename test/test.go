package test

import (
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/vinceprignano/bunny/client"
	"github.com/vinceprignano/bunny/server"
)

type BunnyTest struct {
	suite.Suite
	server *BunnyTestServer
	client *BunnyTestClient
}

func (b *BunnyTest) SetupSuite() {
	b.server = NewBunnyTestServer("bunnytest")
	b.server.On("Init").Return(nil)
	b.server.On("Run").Return(nil)
	b.server.On("RegisterEndpoint", mock.Anything).Return(nil)
	server.NewServer = func(name string) *server.Server {
		b.server.Name = name
		return b.server.(server.Server)
	}
	b.client = NewBunnyTestClient("bunnytest")
	b.client.On("Init").Return(nil)
	client.NewServer = func(name string) *client.Client {
		b.client.Name = name
		return b.client
	}
}
