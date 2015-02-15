package test

import (
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/vinceprignano/bunny"
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

	bunny.NewService = func(name string) *bunny.Service {
		return &bunny.Service{
			Server: b.server,
			Client: b.client,
		}
	}
}
