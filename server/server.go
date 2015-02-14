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

type Server struct {
	Name       string
	registry   *Registry
	connection *rabbit.RabbitConnection
}

var NewServer = func(name string) *Server {
	return &Server{
		Name:       name,
		registry:   NewRegistry(),
		connection: rabbit.NewRabbitConnection(),
	}
}

func (s *Server) Init() {
	select {
	case <-s.connection.Init():
		log.Info("[Server] Connected to RabbitMQ")
	case <-time.After(10 * time.Second):
		log.Critical("[Server] Failed to connect to RabbitMQ")
		os.Exit(1)
	}
}

func (s *Server) RegisterEndpoint(endpoint Endpoint) {
	s.registry.RegisterEndpoint(endpoint)
}

func (s *Server) Run() {
	log.Infof("[Server] Listening for deliveries on %s.#", s.Name)

	deliveries, err := s.connection.Consume(s.Name)
	if err != nil {
		log.Criticalf("[Server] [%s] Failed to consume from Rabbit", s.Name)
	}

	// Range over deliveries from channel
	// This blocks until the channel closes
	for req := range deliveries {
		log.Info("[Server] [%s] Received new delivery", s.Name)
		go s.handleRequest(req)
	}

	log.Infof("Exiting")
	log.Flush()
}

func (s *Server) handleRequest(delivery amqp.Delivery) {
	endpointName := strings.Replace(delivery.RoutingKey, fmt.Sprintf("%s.", s.Name), "", -1)
	endpoint := s.registry.GetEndpoint(endpointName)
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
