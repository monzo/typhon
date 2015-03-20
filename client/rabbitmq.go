// rabbitmq provides a concrete client implementation using
// rabbitmq / amqp as a message bus

package client

import (
	"fmt"
	"os"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/golang/protobuf/proto"
	uuid "github.com/nu7hatch/gouuid"
	"github.com/streadway/amqp"
	"golang.org/x/net/context"

	"github.com/b2aio/typhon/errors"
	pe "github.com/b2aio/typhon/proto/error"
	"github.com/b2aio/typhon/rabbit"
)

var connectionTimeout time.Duration = 10 * time.Second

type RabbitClient struct {
	once       sync.Once
	inflight   *inflightRegistry
	replyTo    string
	connection *rabbit.RabbitConnection
}

var NewRabbitClient = func() Client {
	uuidQueue, err := uuid.NewV4()
	if err != nil {
		log.Criticalf("[Client] Failed to create UUID for reply queue")
		os.Exit(1)
	}
	return &RabbitClient{
		inflight:   newInflightRegistry(),
		connection: rabbit.NewRabbitConnection(),
		replyTo:    fmt.Sprintf("replyTo-%s", uuidQueue.String()),
	}
}

func (c *RabbitClient) Init() {
	select {
	case <-c.connection.Init():
		log.Info("[Client] Connected to RabbitMQ")
	case <-time.After(connectionTimeout):
		log.Critical("[Client] Failed to connect to RabbitMQ after %v", connectionTimeout)
		os.Exit(1)
	}
	c.initConsume()
}

func (c *RabbitClient) initConsume() {
	err := c.connection.Channel.DeclareReplyQueue(c.replyTo)
	if err != nil {
		log.Criticalf("[Client] Failed to declare reply queue: %s", err.Error())
		os.Exit(1)
	}
	deliveries, err := c.connection.Channel.ConsumeQueue(c.replyTo)
	if err != nil {
		log.Criticalf("[Client] Failed to consume from reply queue: %s", err.Error())
		os.Exit(1)
	}
	go func() {
		log.Infof("[Client] Listening for deliveries on %s", c.replyTo)
		for delivery := range deliveries {
			go c.handleDelivery(delivery)
		}
		log.Infof("[Client] Delivery channel %s closed", c.replyTo)
	}()
}

func (c *RabbitClient) handleDelivery(delivery amqp.Delivery) {
	channel := c.inflight.pop(delivery.CorrelationId)
	if channel == nil {
		log.Warnf("[Client] CorrelationID '%s' does not exist in inflight registry", delivery.CorrelationId)
		return
	}
	select {
	case channel <- delivery:
		log.Tracef("[Client] Dispatched delivery to response channel for %s", delivery.CorrelationId)
	default:
		log.Warnf("[Client] Error in delivery for message %s", delivery.CorrelationId)
	}
}

func (c *RabbitClient) Req(ctx context.Context, serviceName, endpoint string, req proto.Message, resp proto.Message) error {

	// Ensure we're initialised, but only do this once
	//
	// @todo we need a connection loop here where we check if we're connected,
	// and if not, block for a short period of time while attempting to reconnect
	c.once.Do(c.Init)

	// Don't even try to send if not connected
	if !c.connection.IsConnected() {
		return errors.Wrap(fmt.Errorf("Not connected to AMQP"))
	}

	routingKey := c.buildRoutingKey(serviceName, endpoint)

	correlation, err := uuid.NewV4()
	if err != nil {
		log.Errorf("[Client] Failed to create unique request id: %v", err)
		return errors.Wrap(err) // @todo custom error code
	}

	log.Debugf("[Client] Dispatching request to %s with correlation ID %s", routingKey, correlation.String())

	replyChannel := c.inflight.push(correlation.String())

	requestBody, err := proto.Marshal(req)
	if err != nil {
		log.Errorf("[Client] Failed to marshal request: %v", err)
		return errors.Wrap(err) // @todo custom error code
	}

	message := amqp.Publishing{
		CorrelationId: correlation.String(),
		Timestamp:     time.Now().UTC(),
		Body:          requestBody,
		ReplyTo:       c.replyTo,
	}

	err = c.connection.Publish(rabbit.Exchange, routingKey, message)
	if err != nil {
		log.Errorf("[Client] Failed to publish %s to '%s': %v", correlation.String(), routingKey, err)
		return errors.Wrap(err) // @todo custom error code
	}

	select {
	case delivery := <-replyChannel:
		log.Debugf("[Client] Response received for %s from %s", correlation.String(), routingKey)
		return handleResponse(delivery, resp)
	case <-time.After(defaultTimeout):
		log.Errorf("[Client] Request %s timed out calling %s", correlation.String(), routingKey)

		return errors.Timeout(fmt.Sprintf("%s timed out", routingKey), nil, map[string]string{
			"called_service":  serviceName,
			"called_endpoint": endpoint,
		})
	}
}

func (c *RabbitClient) CustomReq(req *Request) (Response, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *RabbitClient) buildRoutingKey(serviceName, endpoint string) string {
	return fmt.Sprintf("%s.%s", serviceName, endpoint)
}

// handleResponse returned from a service by marshaling into the response type,
// or converting an error from the remote service
func handleResponse(delivery amqp.Delivery, resp proto.Message) error {

	serverResp := deliveryToResponse(delivery)

	// deal with error responses, by converting back from wire format
	if serverResp.IsError() {
		p := &pe.Error{}
		if err := proto.Unmarshal(serverResp.Payload(), p); err != nil {
			return errors.BadResponse(err.Error())
		}

		return errors.Unmarshal(p)
	}

	// Otherwise try to marshal to the expected response type
	if err := proto.Unmarshal(serverResp.Payload(), resp); err != nil {
		return errors.BadResponse(err.Error())
	}

	return nil
}

// deliveryToResponse converts our AMQP response to a client Response
func deliveryToResponse(delivery amqp.Delivery) *Response {

	contentType, _ := delivery.Headers["Content-Type"].(string)
	contentEncoding, _ := delivery.Headers["Content-Encoding"].(string)
	service, _ := delivery.Headers["Service"].(string)
	endpoint, _ := delivery.Headers["Endpoint"].(string)

	return &response{
		contentType:     contentType,
		contentEncoding: contentEncoding,
		service:         service,
		endpoint:        endpoint,
		payload:         delivery.Body,
	}
}
