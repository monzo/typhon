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
	b.server = NewBunnyTestServer("bunnytest", "a test")
	b.client = NewBunnyTestClient("bunnytest")
	b.server.On("Initialise").Return(nil)
	b.server.On("Run").Return(nil)
	b.server.On("RegisterEndpoint", mock.Anything).Return(nil)
	b.client.On("Initialise").Return(nil)

	server.DefaultServer = b.server

	client.NewRabbitClient = func(name string) client.Client {
		return NewBunnyTestClient(name)
	}
}
