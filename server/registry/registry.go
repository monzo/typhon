package registry

import (
	"sync"

	"github.com/vinceprignano/bunny/transport"
)

type Registry struct {
	sync.Mutex
	endpoints map[string]transport.Endpoint
}

func NewRegistry() *Registry {
	return &Registry{
		endpoints: make(map[string]transport.Endpoint),
	}
}

func (r *Registry) RegisterEndpoint(endpoint transport.Endpoint) {
	r.Lock()
	defer r.Unlock()
	r.endpoints[endpoint.Name()] = endpoint
}

func (r *Registry) GetEndpoint(name string) transport.Endpoint {
	r.Lock()
	defer r.Unlock()
	return r.endpoints[name]
}
