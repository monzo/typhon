package client

import (
	"sync"

	"github.com/streadway/amqp"
)

type dispatcher struct {
	sync.RWMutex
	routes map[string]chan amqp.Delivery
}

func newDispatcher() *dispatcher {
	return &dispatcher{
		routes: make(map[string]chan amqp.Delivery),
	}
}

func (d *dispatcher) add(routingKey string) chan amqp.Delivery {
	d.Lock()
	defer d.Unlock()
	ch := make(chan amqp.Delivery, 1)
	d.routes[routingKey] = ch
	return ch
}

func (d *dispatcher) pop(routingKey string) chan amqp.Delivery {
	d.Lock()
	defer d.Unlock()
	if channel, ok := d.routes[routingKey]; ok {
		delete(d.routes, routingKey)
		return channel
	}
	return nil
}
