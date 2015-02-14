package server

import "sync"

type EndpointRegistry struct {
	sync.Mutex
	endpoints map[string]Endpoint
}

func NewEndpointRegistry() *EndpointRegistry {
	return &EndpointRegistry{
		endpoints: make(map[string]Endpoint),
	}
}

func (r *EndpointRegistry) RegisterEndpoint(endpoint Endpoint) {
	r.Lock()
	defer r.Unlock()
	r.endpoints[endpoint.Name()] = endpoint
}

func (r *EndpointRegistry) GetEndpoint(name string) Endpoint {
	r.Lock()
	defer r.Unlock()
	return r.endpoints[name]
}
