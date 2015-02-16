package server

import (
	"fmt"
	"os"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"github.com/golang/protobuf/proto"
	"github.com/streadway/amqp"
	"github.com/vinceprignano/bunny/rabbit"
)

type Server interface {
	Init()
	Run()
	RegisterEndpoint(endpoint Endpoint)
}

type RabbitServer struct {
	// this is the routing key prefix for all endpoints
	ServiceName      string
	endpointRegistry *EndpointRegistry
	connection       *rabbit.RabbitConnection
}

var NewRabbitServer = func(name string) Server {
	return &RabbitServer{
		ServiceName:      name,
		endpointRegistry: NewEndpointRegistry(),
		connection:       rabbit.NewRabbitConnection(),
	}
}

func (s *RabbitServer) Init() {
	select {
	case <-s.connection.Init():
		log.Info("[Server] Connected to RabbitMQ")
	case <-time.After(10 * time.Second):
		log.Critical("[Server] Failed to connect to RabbitMQ")
		os.Exit(1)
	}
}

func (s *RabbitServer) RegisterEndpoint(endpoint Endpoint) {
	s.endpointRegistry.RegisterEndpoint(endpoint)
}

func (s *RabbitServer) Run() {
	log.Infof("[Server] Listening for deliveries on %s.#", s.ServiceName)

	deliveries, err := s.connection.Consume(s.ServiceName)
	if err != nil {
		log.Criticalf("[Server] [%s] Failed to consume from Rabbit", s.ServiceName)
	}

	// Range over deliveries from channel
	// This blocks until the channel closes
	for req := range deliveries {
		log.Infof("[Server] [%s] Received new delivery", s.ServiceName)
		go s.handleRequest(req)
	}

	log.Infof("Exiting")
	log.Flush()
}

func (s *RabbitServer) handleRequest(delivery amqp.Delivery) {
	// endpointName is everything after the ServiceName. For example if
	// ServiceName = "api.test" and routingKey = "api.test.something.do", then
	// endpointName = "something.do"
	endpointName := strings.Replace(delivery.RoutingKey, fmt.Sprintf("%s.", s.ServiceName), "", -1)
	endpoint := s.endpointRegistry.GetEndpoint(endpointName)
	if endpoint == nil {
		log.Error("[Server] Endpoint not found, cannot handle request")
		return
	}
	req := NewRequest(&delivery)
	rsp, err := endpoint.HandleRequest(req)
	if err != nil {
		log.Errorf("[Server] Endpoint %s returned an error", endpointName)
		log.Error(err.Error())
	}
	body, err := proto.Marshal(rsp)
	if err != nil {
		log.Errorf("[Server] Failed to marshal response")
	}
	msg := amqp.Publishing{
		CorrelationId: delivery.CorrelationId,
		Timestamp:     time.Now().UTC(),
		Body:          body,
	}
	s.connection.Publish("", delivery.ReplyTo, msg)
}
