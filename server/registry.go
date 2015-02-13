package server

import "sync"

type Registry struct {
	sync.Mutex
	endpoints map[string]Endpoint
}

func NewRegistry() *Registry {
	return &Registry{
		endpoints: make(map[string]Endpoint),
	}
}

func (r *Registry) RegisterEndpoint(endpoint Endpoint) {
	r.Lock()
	defer r.Unlock()
	r.endpoints[endpoint.Name()] = endpoint
}

func (r *Registry) GetEndpoint(name string) Endpoint {
	r.Lock()
	defer r.Unlock()
	return r.endpoints[name]
}
