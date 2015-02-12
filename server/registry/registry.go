package registry

import (
	"sync"

	"github.com/vinceprignano/bunny/server/endpoint"
)

type Registry struct {
	sync.Mutex
	endpoints map[string]*endpoint.Endpoint
}

func (r *Registry) RegisterEndpoint(endpoint *endpoint.Endpoint) {
	r.Lock()
	defer r.Unlock()
	r.endpoints[endpoint.Name()] = endpoint
}

func (r *Registry) GetEndpoint(name string) (*endpoint.Endpoint, bool) {
	r.Lock()
	defer r.Unlock()
	return r.endpoints[name]
}
