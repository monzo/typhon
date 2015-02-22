package client

import (
	"sync"

	"github.com/streadway/amqp"
)

type dispatcher struct {
	sync.RWMutex
	requests map[string]chan amqp.Delivery
}

func newDispatcher() *dispatcher {
	return &dispatcher{
		requests: make(map[string]chan amqp.Delivery),
	}
}

func (d *dispatcher) add(requestId string) chan amqp.Delivery {
	d.Lock()
	defer d.Unlock()
	ch := make(chan amqp.Delivery, 1)
	d.requests[requestId] = ch
	return ch
}

func (d *dispatcher) pop(requestId string) chan amqp.Delivery {
	d.Lock()
	defer d.Unlock()
	if channel, ok := d.requests[requestId]; ok {
		delete(d.requests, requestId)
		return channel
	}
	return nil
}
