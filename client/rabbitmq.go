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

	"github.com/mondough/typhon/errors"
	"github.com/mondough/typhon/rabbit"
	"github.com/mondough/typhon/server"

	pe "github.com/mondough/typhon/proto/error"
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

func (c *RabbitClient) Req(ctx context.Context, service, endpoint string, req proto.Message, resp proto.Message) error {

	// Build request
	payload, err := proto.Marshal(req)
	if err != nil {
		log.Errorf("[Client] Failed to marshal request: %v", err)
		return errors.Wrap(err) // @todo custom error code
	}
	protoReq, err := NewProtoRequest(service, endpoint, payload)
	if err != nil {
		return errors.Wrap(err)
	}

	// Execute
	rsp, err := c.do(ctx, protoReq)
	if err != nil {
		return err
	}

	// Unmarshal response into the provided pointer
	if err := unmarshalResponse(rsp, resp); err != nil {
		return err
	}

	return nil
}

// CustomReq makes a sends a request to a service and returns a
// response without the usual marshaling helpers
func (c *RabbitClient) CustomReq(ctx context.Context, req Request) (Response, error) {
	return c.do(ctx, req)
}

// do sends a request and returns a response, following policies
// (e.g. redirects, cookies, auth) as configured on the client.
func (c *RabbitClient) do(ctx context.Context, req Request) (Response, error) {

	// Ensure we're initialised, but only do this once
	//
	// @todo we need a connection loop here where we check if we're connected,
	// and if not, block for a short period of time while attempting to reconnect
	c.once.Do(c.Init)

	// Don't even try to send if not connected
	if !c.connection.IsConnected() {
		return nil, errors.Wrap(fmt.Errorf("Not connected to AMQP"))
	}

	// Push the reply channel into our request registry
	replyChannel := c.inflight.push(req.Id())

	routingKey := req.Service()

	// Build message from request
	message := amqp.Publishing{
		CorrelationId: req.Id(),
		Timestamp:     time.Now().UTC(),
		Body:          req.Payload(),
		ReplyTo:       c.replyTo,
		Headers: amqp.Table{
			"Content-Type":     req.ContentType(),
			"Content-Encoding": "request",
			"Service":          req.Service(),
			"Endpoint":         req.Endpoint(),
		},
	}

	parentRequest := server.RecoverRequestFromContext(ctx)

	// if there's a parent request, set the parent request id
	// and trace ID from it. If not, generate a new trace ID
	if parentRequest != nil {
		message.Headers["Parent-Request-ID"] = parentRequest.Id()
		message.Headers["Trace-ID"] = parentRequest.TraceID()
	} else {
		traceID, err := uuid.NewV4()
		if err != nil {
			return nil, err
		}
		// @todo the trace id format could be doner nicer
		message.Headers["Trace-ID"] = "trace_" + traceID.String()
	}

	// Look for access token on the request first, context second
	accessToken := req.AccessToken()
	if accessToken == "" && parentRequest != nil {
		accessToken = parentRequest.AccessToken()
	}
	if accessToken != "" {
		message.Headers["Access-Token"] = accessToken
	}

	// If we have a recovered session for the parent request, set it on the child request
	// so that the called service doesn't have to call RecoverSession() again
	if parentRequest != nil && parentRequest.HasRecoveredSession() &&
		parentRequest.Server().AuthenticationProvider() != nil {

		session, err := parentRequest.Session()
		if err != nil {
			return nil, err
		}

		sessionBytes, err := parentRequest.Server().AuthenticationProvider().MarshalSession(session)
		if err != nil {
			log.Warnf("[Client] Failed to marshal recovered session") // @todo log somewhere
		} else {
			message.Headers["Session"] = sessionBytes
		}
	}

	log.Debugf("[Client] Dispatching request to %s with correlation ID %s, reply to %s and headers %+v", routingKey, req.Id(), c.replyTo, message.Headers)

	// Attempt to publish through our connection
	// @todo refactor this to not know about rabbitmq internals
	err := c.connection.Publish(rabbit.Exchange, routingKey, message)
	if err != nil {
		log.Errorf("[Client] Failed to publish %s to '%s': %v", req.Id(), routingKey, err)

		// Remove from inflight requests
		_ = c.inflight.pop(req.Id())

		return nil, errors.Wrap(err) // @todo custom error code
	}

	select {
	case delivery := <-replyChannel:
		log.Debugf("[Client] Response received for %s from %s", req.Id(), routingKey)
		rsp := deliveryToResponse(delivery)
		if rsp.IsError() {
			return nil, unmarshalErrorResponse(rsp)
		}
		return rsp, nil
	case <-time.After(req.Timeout()):
		log.Errorf("[Client] Request %s timed out calling %s", req.Id(), routingKey)

		return nil, errors.Timeout(fmt.Sprintf("%s timed out", routingKey), nil, map[string]string{
			"called_service":  req.Service(),
			"called_endpoint": req.Endpoint(),
		})
	}

}

// unmarshalResponse returned from a service into the response type
func unmarshalResponse(resp Response, respProto proto.Message) error {
	if err := proto.Unmarshal(resp.Payload(), respProto); err != nil {
		return errors.BadResponse(err.Error())
	}

	return nil
}

// deliveryToResponse converts our AMQP response to a client Response
func deliveryToResponse(delivery amqp.Delivery) Response {

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

// unmarshalErrorResponse from our wire format to a typhon error
func unmarshalErrorResponse(resp Response) error {
	p := &pe.Error{}
	if err := proto.Unmarshal(resp.Payload(), p); err != nil {
		return errors.BadResponse(err.Error())
	}

	return errors.Unmarshal(p)
}
