package test

import (
	"github.com/b2aio/typhon/server"
	"github.com/golang/protobuf/proto"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/suite"
)

type RabbitTestSuite struct {
	suite.Suite
	stubServer *StubServer
}

// `TearDownTest` is run after each test and tears down stubbed responses
func (t *RabbitTestSuite) TearDownTest() {
	if t.stubServer != nil {
		t.stubServer.ResetStubs()
	}
}

// Stubs a service call so that when `endpoint` is called on `serviceName`,
// `returnValue` is returned
func (t *RabbitTestSuite) StubResponse(serviceName, endpoint string, returnValue proto.Message) {
	t.StubResponseWithError(serviceName, endpoint, returnValue, nil)
}

// Stubs an error
func (t *RabbitTestSuite) StubError(serviceName, endpoint string, err error) {
	t.StubResponseWithError(serviceName, endpoint, nil, err)
}

// Registers a stub response with the stubServer
func (t *RabbitTestSuite) StubResponseWithError(serviceName, endpoint string, returnValue proto.Message, err error) {
	t.lazyStubServer().RegisterStub(&ServiceStub{
		ServiceName: serviceName,
		Endpoint:    endpoint,
		Handler: func(_ server.Request) (server.Response, error) {
			return server.NewProtoResponse(returnValue), err
		},
	})
}

// Helper method to call a handler function (the function being tested)
// directly with a `proto.Message`.
// Returns errors that were returned from the handler function directly.
// Marshalling errors cause the test to fail instantly
func (t *RabbitTestSuite) CallHandler(handler server.HandlerFunc, reqProto proto.Message, respProto proto.Message) error {
	// Call handler with amqp delivery
	reqBytes, err := proto.Marshal(reqProto)
	t.NoError(err)
	resp, err := handler(server.NewAMQPRequest(&amqp.Delivery{
		Body: reqBytes,
	}))
	if err != nil {
		return err
	}
	respBytes, err := resp.Encode()
	t.NoError(err)
	err = proto.Unmarshal(respBytes, respProto)
	t.NoError(err)
	return nil
}

// Lazy initialize `t.stubServer`
func (t *RabbitTestSuite) lazyStubServer() *StubServer {
	if t.stubServer == nil {
		t.stubServer = NewStubServer(t)
	}
	return t.stubServer
}
