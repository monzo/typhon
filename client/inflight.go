package client

import (
	"sync"

	"github.com/streadway/amqp"
)

// inflightRegistry is a registry that keeps track of requests which are currently
// inflight to other services, with a channel back to the originating client
type inflightRegistry struct {
	sync.Mutex
	requests map[string]chan amqp.Delivery
}

// newInflightRegistry creates an initialised inflight registry
func newInflightRegistry() *inflightRegistry {
	return &inflightRegistry{
		requests: make(map[string]chan amqp.Delivery),
	}
}

// push a request onto the stack
func (r *inflightRegistry) push(requestId string) chan amqp.Delivery {
	r.Lock()
	ch := make(chan amqp.Delivery, 1)
	r.requests[requestId] = ch
	r.Unlock()
	return ch
}

// pop a request off the stack
func (r *inflightRegistry) pop(requestId string) chan amqp.Delivery {
	r.Lock()
	defer r.Unlock()
	if channel, ok := r.requests[requestId]; ok {
		delete(r.requests, requestId)
		return channel
	}
	return nil
}
