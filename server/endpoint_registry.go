package server

import (
	"regexp"
	"sync"

	"github.com/b2aio/typhon/auth"
)

type EndpointRegistry struct {
	sync.RWMutex
	endpoints map[string]*Endpoint
}

func NewEndpointRegistry() *EndpointRegistry {
	return &EndpointRegistry{
		endpoints: make(map[string]*Endpoint),
	}
}

func (r *EndpointRegistry) Get(endpointName string) *Endpoint {
	r.RLock()
	defer r.RUnlock()
	for pattern, endpoint := range r.endpoints {
		if match, _ := regexp.Match("^"+pattern+"$", []byte(endpointName)); match == true {
			return endpoint
		}
	}
	return nil
}

// Register an endpoint with the registry
func (r *EndpointRegistry) Register(e *Endpoint) {

	// Always set an Authorizer on an endpoint
	if e.Authorizer == nil {
		e.Authorizer = auth.DefaultAuthorizer
	}

	r.Lock()
	defer r.Unlock()
	r.endpoints[e.Name] = e
}

func (r *EndpointRegistry) Deregister(pattern string) {
	r.Lock()
	defer r.Unlock()
	delete(r.endpoints, pattern)
}
