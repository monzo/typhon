// rabbitmq provides a concrete client implementation using
// rabbitmq / amqp as a message bus

package client

import (
	"errors"
	"fmt"
	"os"
	"time"

	log "github.com/cihub/seelog"
	"github.com/golang/protobuf/proto"
	uuid "github.com/nu7hatch/gouuid"
	"github.com/streadway/amqp"

	"github.com/b2aio/typhon/rabbit"
)

var (
	ErrBadResponse    = errors.New("client.unmarshal.badreply.error")
	ErrPublish        = errors.New("client.call.publish.error")
	ErrRequestMarshal = errors.New("client.call.marshal.error")
	ErrUuidGeneration = errors.New("client.call.uuid.error")
	ErrTimeout        = errors.New("client.call.timeout.error")
)

type RabbitClient struct {
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
	case <-time.After(10 * time.Second):
		log.Critical("[Client] Failed to connect to RabbitMQ")
		os.Exit(1)
	}
	c.initConsume()
}

func (c *RabbitClient) initConsume() {
	err := c.connection.Channel.DeclareReplyQueue(c.replyTo)
	if err != nil {
		log.Critical("[Client] Failed to declare reply queue")
		log.Critical(err.Error())
		os.Exit(1)
	}
	deliveries, err := c.connection.Channel.ConsumeQueue(c.replyTo)
	if err != nil {
		log.Critical("[Client] Failed to consume from reply queue")
		log.Critical(err.Error())
		os.Exit(1)
	}
	go func() {
		log.Infof("[Client] Listening for deliveries on %s", c.replyTo)
		for delivery := range deliveries {
			go c.handleDelivery(delivery)
		}
	}()
}

func (c *RabbitClient) handleDelivery(delivery amqp.Delivery) {
	channel := c.inflight.pop(delivery.CorrelationId)
	if channel == nil {
		log.Errorf("[Client] CorrelationID '%s' does not exist in inflight registry", delivery.CorrelationId)
		return
	}
	select {
	case channel <- delivery:
	default:
		log.Errorf("[Client] Error in delivery for correlation %s", delivery.CorrelationId)
	}
}

func (c *RabbitClient) Call(serviceName, endpoint string, req proto.Message, res proto.Message) error {

	routingKey := c.buildRoutingKey(serviceName, endpoint)

	correlation, err := uuid.NewV4()
	if err != nil {
		log.Errorf("[Client] Failed to create unique request id: %v", err)
		return ErrUuidGeneration
	}

	replyChannel := c.inflight.push(correlation.String())
	defer close(replyChannel)

	requestBody, err := proto.Marshal(req)
	if err != nil {
		log.Errorf("[Client] Failed to marshal request: %v", err)
		return ErrRequestMarshal
	}

	message := amqp.Publishing{
		CorrelationId: correlation.String(),
		Timestamp:     time.Now().UTC(),
		Body:          requestBody,
		ReplyTo:       c.replyTo,
	}

	err = c.connection.Publish(rabbit.Exchange, routingKey, message)
	if err != nil {
		log.Errorf("[Client] Failed to publish to '%s': %v", routingKey, err)
		return ErrPublish
	}

	select {
	case delivery := <-replyChannel:
		if err := proto.Unmarshal(delivery.Body, res); err != nil {
			return ErrBadResponse
		}
		return nil
	case <-time.After(defaultTimeout):
		log.Criticalf("[Client] Client timeout on delivery")
		return ErrTimeout
	}
}

func (c *RabbitClient) buildRoutingKey(serviceName, endpoint string) string {
	return fmt.Sprintf("%s.%s", serviceName, endpoint)
}
