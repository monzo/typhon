package server

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"github.com/golang/protobuf/proto"
	"github.com/streadway/amqp"

	"github.com/vinceprignano/bunny/rabbit"
)

type AMQPServer struct {
	// this is the routing key prefix for all endpoints
	ServiceName        string
	ServiceDescription string
	endpointRegistry   *EndpointRegistry
	connection         *rabbit.RabbitConnection
}

func NewAMQPServer() Server {
	return &AMQPServer{
		endpointRegistry: NewEndpointRegistry(),
		connection:       rabbit.NewRabbitConnection(),
	}
}

func (s *AMQPServer) Name() string {
	if s == nil {
		return ""
	}
	return s.ServiceName
}

func (s *AMQPServer) Description() string {
	if s == nil {
		return ""
	}
	return s.ServiceDescription
}

func (s *AMQPServer) Init(c *Config) {
	s.ServiceName = c.Name
	s.ServiceDescription = c.Description
}

func (s *AMQPServer) RegisterEndpoint(endpoint Endpoint) {
	s.endpointRegistry.Register(endpoint)
}

func (s *AMQPServer) DeregisterEndpoint(endpointName string) {
	s.endpointRegistry.Deregister(endpointName)
}

// Run the server, connecting to our transport and serving requests
func (s *AMQPServer) Run() {

	// Connect to AMQP
	select {
	case <-s.connection.Init():
		log.Info("[Server] Connected to RabbitMQ")
	case <-time.After(10 * time.Second):
		log.Critical("[Server] Failed to connect to RabbitMQ")
		os.Exit(1)
	}

	// Get a delivery channel from the connection
	log.Infof("[Server] Listening for deliveries on %s.#", s.ServiceName)
	deliveries, err := s.connection.Consume(s.ServiceName)
	if err != nil {
		log.Criticalf("[Server] [%s] Failed to consume from Rabbit", s.ServiceName)
	}

	// Handle deliveries
	for req := range deliveries {
		log.Infof("[Server] [%s] Received new delivery", s.ServiceName)
		go s.handleRequest(req)
	}

	log.Infof("Exiting")
	log.Flush()
}

// handleRequest takes a delivery from AMQP, attempts to process it and return a response
func (s *AMQPServer) handleRequest(delivery amqp.Delivery) {

	// See if we have a matching endpoint for this request
	endpointName := strings.Replace(delivery.RoutingKey, fmt.Sprintf("%s.", s.ServiceName), "", -1)
	endpoint := s.endpointRegistry.Get(endpointName)
	if endpoint == nil {
		log.Error("[Server] Endpoint not found, cannot handle request")
		s.respondWithError(delivery, errors.New("Endpoint not found"))
		return
	}

	// Handle the delivery
	req := NewAMQPRequest(&delivery)
	rsp, err := endpoint.HandleRequest(req)
	if err != nil {
		log.Errorf("[Server] Endpoint %s returned an error", endpointName)
		log.Error(err.Error())
	}

	// Marshal the response
	body, err := proto.Marshal(rsp)
	if err != nil {
		log.Errorf("[Server] Failed to marshal response")
	}

	// Build return delivery, and publish
	msg := amqp.Publishing{
		CorrelationId: delivery.CorrelationId,
		Timestamp:     time.Now().UTC(),
		Body:          body,
	}
	s.connection.Publish("", delivery.ReplyTo, msg)
}

// respondWithError to a delivery, with the provided error
func (s *AMQPServer) respondWithError(delivery amqp.Delivery, err error) {

	// Construct a return message with an error
	msg := amqp.Publishing{
		CorrelationId: delivery.CorrelationId,
		Timestamp:     time.Now().UTC(),
		Body:          []byte(err.Error()),
	}

	// Publish the error back to the client
	s.connection.Publish("", delivery.ReplyTo, msg)
}
