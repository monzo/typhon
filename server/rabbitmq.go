package server

import (
	"encoding/json"
	"time"

	log "github.com/cihub/seelog"
	"github.com/golang/protobuf/proto"
	"github.com/streadway/amqp"

	"github.com/b2aio/typhon/errors"
	"github.com/b2aio/typhon/rabbit"
)

var connectionTimeout time.Duration = 10 * time.Second

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

func (s *AMQPServer) RegisterEndpoint(endpoint *Endpoint) {
	s.endpointRegistry.Register(endpoint)
}

func (s *AMQPServer) DeregisterEndpoint(endpointName string) {
	s.endpointRegistry.Deregister(endpointName)
}

// Run the server, connecting to our transport and serving requests
func (s *AMQPServer) Run() {
	defer log.Flush()

	// Connect to AMQP
	select {
	case <-s.connection.Init():
		log.Info("[Server] Connected to RabbitMQ")
	case <-time.After(connectionTimeout):
		log.Critical("[Server] Failed to connect to RabbitMQ after %v", connectionTimeout)
		return
	}

	// Get a delivery channel from the connection
	log.Infof("[Server] Listening for deliveries on %s.#", s.ServiceName)
	deliveries, err := s.connection.Consume(s.ServiceName)
	if err != nil {
		log.Infof("[Server] Failed to consume from Rabbit: %s", err.Error())
		return
	}

	// Notify observers that we are ready to consume
	for _, notify := range s.notifyConnected {
		select {
		case notify <- true:
		default:
		}
	}

	// Handle deliveries
	for {
		select {
		case req, ok := <-deliveries:
			if !ok {
				log.Infof("[Server] Delivery channel closed, exiting")
				return
			}
			log.Tracef("[Server] Received new delivery: %#v", req)
			go s.handleDelivery(req)
		case <-s.closeChan:
			// shut down server
			log.Infof("[Server] Closing connection")
			s.connection.Close()
			log.Infof("[Server] Connection closed")
			return
		}
	}
}

func (s *AMQPServer) Close() {
	close(s.closeChan)
}

// handleDelivery takes a delivery from AMQP, attempts to process it and return a response
func (s *AMQPServer) handleDelivery(delivery amqp.Delivery) {
	log.Tracef("Handling Request (delivery): %s", delivery.RoutingKey)
	var err error

	// Marshal to a request
	req := NewAMQPRequest(&delivery)

	// See if we have a matching endpoint for this request
	endpoint := s.endpointRegistry.Get(req.Endpoint())
	if endpoint == nil {
		log.Errorf("[Server] Endpoint '%s' not found, cannot handle request", req.Endpoint())
		s.respondWithError(delivery, errors.BadRequest("Endpoint not found"))
		return
	}

	// Handle the delivery
	resp, err := endpoint.HandleRequest(req)
	if err != nil {
		log.Warnf("[Server] Failed to handle request: %s", err.Error())
		s.respondWithError(delivery, err)
		return
	}
	if resp == nil {
		s.respondWithError(delivery, errors.BadResponse("Handler returned nil"))
		return
	}

	// Marshal the response
	// @todo we're currently always marshaling errors as proto
	var body []byte
	if req.ContentType() == "application/x-protobuf" {
		body, err = proto.Marshal(resp)
	} else {
		body, err = json.Marshal(resp)
	}
	if err != nil {
		log.Errorf("[Server] Failed to marshal response: %s", err.Error())
		s.respondWithError(delivery, errors.BadResponse("Failed to marshal response: "+err.Error()))
		return
	}

	// Build return delivery, and publish
	msg := amqp.Publishing{
		CorrelationId: req.Id(),
		Timestamp:     time.Now().UTC(),
		Body:          body,
		Headers: amqp.Table{
			"Content-Type":     req.ContentType(),
			"Content-Encoding": "response",
			"Service":          req.Service(),
			"Endpoint":         req.Endpoint(),
		},
	}

	log.Tracef("[Server] Sending response to %s", delivery.ReplyTo)
	s.connection.Publish("", delivery.ReplyTo, msg)
}

// respondWithError to a delivery, with the provided error
func (s *AMQPServer) respondWithError(delivery amqp.Delivery, err error) {

	// Ensure we have a service error in proto form
	// and marshal this for transmission
	svcErr := errors.Wrap(err)
	b, err := proto.Marshal(errors.Marshal(svcErr))
	if err != nil {
		// shit
	}

	// Construct a return message with an error
	msg := amqp.Publishing{
		CorrelationId: delivery.CorrelationId,
		Timestamp:     time.Now().UTC(),
		Body:          b,
		Headers: amqp.Table{
			"Content-Type":     "application/x-protobuf",
			"Content-Encoding": "error",
		},
	}

	// Publish the error back to the client
	log.Tracef("[Server] Sending error response to %s", delivery.ReplyTo)
	s.connection.Publish("", delivery.ReplyTo, msg)
}
