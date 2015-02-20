package test

import (
	"fmt"
	"time"

	"github.com/b2aio/typhon/client"
	"github.com/b2aio/typhon/server"
	"github.com/golang/protobuf/proto"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/suite"
)

type RabbitTestSuite struct {
	suite.Suite
	Server server.Server
}

func (t *RabbitTestSuite) SetupSuite() {
	t.Server = server.NewAMQPServer()
	t.Server.Init(&server.Config{Name: "#", Description: "Test Server"})
	go t.Server.Run()
	select {
	case <-t.Server.NotifyConnected():
	case <-time.After(10 * time.Second):
		t.T().FailNow()
	}
	client.InitDefault("testing")
}

func (t *RabbitTestSuite) StubEndpoint(endpoint string, returnValue server.Response, returnError error) {
	t.Server.RegisterEndpoint(&server.DefaultEndpoint{
		EndpointName: fmt.Sprintf("%s", endpoint),
		Handler:      wrapHandler(returnValue, returnError),
	})
}

func (r *RabbitTestSuite) NewRequestFromProto(message proto.Message) server.Request {
	body, err := proto.Marshal(message)
	r.NoError(err)
	return server.NewAMQPRequest(&amqp.Delivery{
		Body: body,
	})
}

type handlerFunc func(request server.Request) (server.Response, error)

func wrapHandler(returnValue server.Response, returnError error) handlerFunc {
	return func(request server.Request) (server.Response, error) {
		if returnError != nil {
			return nil, returnError
		}
		return returnValue, nil
	}
}
