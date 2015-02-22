package client

import (
	"sync"

	"github.com/streadway/amqp"
)

type inflightRegistry struct {
	sync.RWMutex
	requests map[string]chan amqp.Delivery
}

func newInflightRegistry() *inflightRegistry {
	return &inflightRegistry{
		requests: make(map[string]chan amqp.Delivery),
	}
}

func (r *inflightRegistry) add(requestId string) chan amqp.Delivery {
	r.Lock()
	defer r.Unlock()
	ch := make(chan amqp.Delivery, 1)
	r.requests[requestId] = ch
	return ch
}

func (r *inflightRegistry) pop(requestId string) chan amqp.Delivery {
	r.Lock()
	defer r.Unlock()
	if channel, ok := r.requests[requestId]; ok {
		delete(r.requests, requestId)
		return channel
	}
	return nil
}
