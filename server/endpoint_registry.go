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

func (r *EndpointRegistry) Get(pattern string) Endpoint {
	r.Lock()
	defer r.Unlock()
	return r.endpoints[pattern]
}

func (r *EndpointRegistry) Register(endpoint Endpoint) {
	r.Lock()
	defer r.Unlock()
	r.endpoints[endpoint.Name()] = endpoint
}

func (r *EndpointRegistry) Deregister(pattern string) {
	r.Lock()
	defer r.Unlock()
	delete(r.endpoints, pattern)
}
