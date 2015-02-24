package server

import (
	"regexp"
	"sync"
)

type EndpointRegistry struct {
	sync.RWMutex
	endpoints map[string]Endpoint
}

func NewEndpointRegistry() *EndpointRegistry {
	return &EndpointRegistry{
		endpoints: make(map[string]Endpoint),
	}
}

func (r *EndpointRegistry) Get(endpointName string) Endpoint {
	r.RLock()
	defer r.RUnlock()
	for pattern, endpoint := range r.endpoints {
		if match, _ := regexp.Match("^"+pattern+"$", []byte(endpointName)); match == true {
			return endpoint
		}
	}
	return nil
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
