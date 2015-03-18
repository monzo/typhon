package test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/b2aio/typhon/server"
)

type StubServer struct {
	server.Server
	t *testing.T

	stubsMutex sync.RWMutex
	stubs      []*ServiceStub
}

type ServiceStub struct {
	ServiceName string
	Endpoint    string
	Handler     func(server.Request) (proto.Message, error)
}

// The stub server boots up a regular typhon server and registers a single
// endpoint that subscribes to every routing key
func NewStubServer(t *testing.T) *StubServer {

	stubServer := &StubServer{
		Server: server.NewAMQPServer(),
		t:      t,
	}

	// TODO `Name: "#"` is rather hokey
	stubServer.Init(&server.Config{Name: "#", Description: "Stub Server"})

	go stubServer.Run()

	select {
	case <-stubServer.NotifyConnected():
	case <-time.After(1 * time.Second):
		t.Fatalf("StubServer couldn't connect to RabbitMQ")
	}

	t.Log("[StubServer] Connected to RabbitMQ")

	stubServer.RegisterEndpoint(&server.Endpoint{
		Name: ".*", // TODO Name is not well-named
		Handler: func(req server.Request) (proto.Message, error) {
			return stubServer.handleRequest(req)
		},
	})

	return stubServer
}

// StubResponse is a convenience method to quickly set up stubs that return a fixed value
func (stubServer *StubServer) StubResponse(serviceName, endpoint string, returnValue proto.Message) {
	stubServer.stubResponseAndError(serviceName, endpoint, returnValue, nil)
}

// StubError is a convenience method to stub out a service error
func (stubServer *StubServer) StubError(serviceName, endpoint string, err error) {
	stubServer.stubResponseAndError(serviceName, endpoint, nil, err)
}

// stubResponseAndError registers a stub that returns the passed response and error
func (stubServer *StubServer) stubResponseAndError(serviceName, endpoint string, returnValue proto.Message, err error) {
	stubServer.RegisterStub(&ServiceStub{
		ServiceName: serviceName,
		Endpoint:    endpoint,
		Handler: func(_ server.Request) (proto.Message, error) {
			return returnValue, err
		},
	})
}

// Registers a stub with the server
func (stubServer *StubServer) RegisterStub(stub *ServiceStub) {
	stubServer.stubsMutex.Lock()
	stubServer.t.Logf("[StubServer] Registered stub for %s.%s", stub.ServiceName, stub.Endpoint)
	defer stubServer.stubsMutex.Unlock()
	stubServer.stubs = append(stubServer.stubs, stub)
}

// Clear out all server stubs. Test suites should run this between tests
func (stubServer *StubServer) ResetStubs() {
	stubServer.stubsMutex.Lock()
	defer stubServer.stubsMutex.Unlock()
	stubServer.stubs = nil
	stubServer.t.Log("[StubServer] Stubs cleared")
}

// Finds the relevant endpoint stub (if any), and calls its handler function
func (stubServer *StubServer) handleRequest(req server.Request) (proto.Message, error) {

	stubServer.t.Logf("[StubServer] Handling request for %s", req.Service(), req.Endpoint())
	stubServer.stubsMutex.RLock()
	defer stubServer.stubsMutex.RUnlock()

	// determine which endpoint to use
	for _, stub := range stubServer.stubs {
		if stub.ServiceName == req.Service() && stub.Endpoint == req.Endpoint() {
			return stub.Handler(req)
		}
	}
	return nil, fmt.Errorf("No stub found for routing service name %s and endpoint %s", req.Service(), req.Endpoint())
}
