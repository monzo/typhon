package server

import (
	"regexp"
	"sync"

	"github.com/b2aio/typhon/auth"
)

// EndpointRegistry stores a list of endpoints for the server
type EndpointRegistry struct {
	sync.RWMutex
	endpoints map[string]*Endpoint
}

// NewEndpointRegistry returns an initialised endpoint registry
func NewEndpointRegistry() *EndpointRegistry {
	return &EndpointRegistry{
		endpoints: make(map[string]*Endpoint),
	}
}

// Get an endpoint with a matching name
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

// Deregister and endpoint from the registry
func (r *EndpointRegistry) Deregister(pattern string) {
	r.Lock()
	defer r.Unlock()
	delete(r.endpoints, pattern)
}
