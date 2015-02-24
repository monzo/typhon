package test

import (
	"fmt"
	"sync"
	"time"

	"github.com/b2aio/typhon/server"
)

// Test cases can register stubs with this StubServer via the helpers
// in `RabbitTestSuite`.
type StubServer struct {
	server.Server
	testSuite *RabbitTestSuite

	stubsMutex sync.RWMutex
	stubs      []*ServiceStub
}

type ServiceStub struct {
	ServiceName string
	Endpoint    string
	Handler     server.HandlerFunc
}

// The stub server boots up a regular typhon server and registers a single
// endpoint that subscribes to every routing key
func NewStubServer(testSuite *RabbitTestSuite) *StubServer {

	stubServer := &StubServer{
		Server:    server.NewAMQPServer(),
		testSuite: testSuite,
	}

	// TODO `Name: "#"` is rather hokey
	stubServer.Init(&server.Config{Name: "#", Description: "Stub Server"})

	go stubServer.Run()

	select {
	case <-stubServer.NotifyConnected():
	case <-time.After(1 * time.Second):
		stubServer.testSuite.T().Fatalf("StubServer couldn't connect to RabbitMQ")
	}

	stubServer.RegisterEndpoint(&server.DefaultEndpoint{
		EndpointName: ".*", // TODO EndpointName is not well-named
		Handler: func(req server.Request) (server.Response, error) {
			return stubServer.handleRequest(req)
		},
	})

	return stubServer
}

// Finds the relevant endpoint stub (if any), and calls its handler function
func (stubServer *StubServer) handleRequest(req server.Request) (server.Response, error) {
	stubServer.stubsMutex.RLock()
	defer stubServer.stubsMutex.RUnlock()

	// determine which endpoint to use
	for _, stub := range stubServer.stubs {
		if stub.ServiceName == req.ServiceName() && stub.Endpoint == req.Endpoint() {
			return stub.Handler(req)
		}
	}

	return nil, fmt.Errorf("No stub found for routing service name %s and endpoint %s", req.ServiceName(), req.Endpoint())
}

// Registers a stub with the server
func (stubServer *StubServer) RegisterStub(stub *ServiceStub) {
	stubServer.stubsMutex.Lock()
	defer stubServer.stubsMutex.Unlock()
	stubServer.stubs = append(stubServer.stubs, stub)
}

// Clear out all server stubs. Test suites should run this between tests
func (stubServer *StubServer) ResetStubs() {
	stubServer.stubsMutex.Lock()
	defer stubServer.stubsMutex.Unlock()
	stubServer.stubs = nil
}
