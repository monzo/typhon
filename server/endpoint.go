package server

import "github.com/vinceprignano/bunny/transport"

type Endpoint interface {
	Name() string
	HandleRequest(req transport.Request) ([]byte, error)
}

type DefaultEndpoint struct {
	EndpointName string
	Handler      func(req transport.Request) ([]byte, error)
}

func (d *DefaultEndpoint) Name() string {
	return d.EndpointName
}

func (d *DefaultEndpoint) HandleRequest(req transport.Request) ([]byte, error) {
	return d.Handler(req)
}
