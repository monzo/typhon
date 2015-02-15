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
	b.client = NewBunnyTestClient("bunnytest")
	b.server.On("Init").Return(nil)
	b.server.On("Run").Return(nil)
	b.server.On("RegisterEndpoint", mock.Anything).Return(nil)
	b.client.On("Init").Return(nil)

	server.NewRabbitServer = func(name string) Server {
		return NewBunnyTestServer(name)
	}

	client.NewRabbitClient = func(name string) Client {
		return NewBunnyTestClient(name)
	}
}
