package server

import (
	"fmt"
	"os"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"github.com/golang/protobuf/proto"
	"github.com/streadway/amqp"

	"github.com/b2aio/typhon/errors"
	"github.com/b2aio/typhon/rabbit"
)

type AMQPServer struct {
	// this is the routing key prefix for all endpoints
	ServiceName        string
	ServiceDescription string
	endpointRegistry   *EndpointRegistry
	connection         *rabbit.RabbitConnection
	notifyConnected    []chan bool

	closeChan chan struct{}
}

func NewAMQPServer() Server {
	return &AMQPServer{
		endpointRegistry: NewEndpointRegistry(),
		connection:       rabbit.NewRabbitConnection(),
		closeChan:        make(chan struct{}),
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

func (s *AMQPServer) NotifyConnected() chan bool {
	ch := make(chan bool)
	s.notifyConnected = append(s.notifyConnected, ch)
	return ch
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
		for _, notify := range s.notifyConnected {
			notify <- true
		}
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
	for {
		select {
		case req := <-deliveries:
			log.Infof("[Server] [%s] Received new delivery", s.ServiceName)
			go s.handleRequest(req)
		case <-s.closeChan:
			// shut down server
			s.connection.Close()
			break
		}
	}

	log.Infof("Exiting")
	log.Flush()
}

func (s *AMQPServer) Close() {
	close(s.closeChan)
}

// handleRequest takes a delivery from AMQP, attempts to process it and return a response
func (s *AMQPServer) handleRequest(delivery amqp.Delivery) {

	// See if we have a matching endpoint for this request
	endpointName := strings.Replace(delivery.RoutingKey, fmt.Sprintf("%s.", s.ServiceName), "", -1)
	endpoint := s.endpointRegistry.Get(endpointName)
	if endpoint == nil {
		log.Errorf("[Server] Endpoint '%s' not found, cannot handle request", endpointName)
		s.respondWithError(delivery, errors.BadRequest("endpoint.notfound", "Endpoint not found"))
		return
	}

	// Handle the delivery
	req := NewAMQPRequest(&delivery)
	rsp, err := endpoint.HandleRequest(req)
	if err != nil {
		s.respondWithError(delivery, err)
		return
	}

	// TODO deal with rsp == nil (programmer error, but still)

	// Marshal the response
	body, err := rsp.Encode()
	if err != nil {
		log.Errorf("[Server] Failed to marshal response")
		s.respondWithError(delivery, errors.BadResponse("response.marshal", err.Error()))
		return
	}

	// Build return delivery, and publish
	msg := amqp.Publishing{
		CorrelationId: delivery.CorrelationId,
		Timestamp:     time.Now().UTC(),
		Body:          body,
		Headers: map[string]interface{}{
			"Content-Encoding": "RESPONSE",
		},
	}
	s.connection.Publish("", delivery.ReplyTo, msg)
}

// respondWithError to a delivery, with the provided error
func (s *AMQPServer) respondWithError(delivery amqp.Delivery, err error) {

	// Ensure we have a service error in proto form
	// and marshal this for transmission
	svcErr := errorToServiceError(err)
	b, err := proto.Marshal(errors.Marshal(svcErr))
	if err != nil {
		// shit
	}

	// Construct a return message with an error
	msg := amqp.Publishing{
		CorrelationId: delivery.CorrelationId,
		Timestamp:     time.Now().UTC(),
		Body:          b,
		Headers: map[string]interface{}{
			"Content-Encoding": "ERROR",
		},
	}

	// Publish the error back to the client
	s.connection.Publish("", delivery.ReplyTo, msg)
}

// errorToServiceError converts an error interface to the concrete
// errors.ServiceError type which we pass between services (as proto)
func errorToServiceError(err error) *errors.ServiceError {

	// If we already have a service error, return this
	if svcErr, ok := err.(*errors.ServiceError); ok {
		return svcErr
	}

	// Otherwise make a generic internal service error
	// Encapsulating the error we were given
	return errors.InternalService("internalservice", err.Error()).(*errors.ServiceError)
}
