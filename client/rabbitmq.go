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

type RabbitClient struct {
	Name       string
	inflight   *inflightRegistry
	replyTo    string
	connection *rabbit.RabbitConnection
}

var NewRabbitClient = func(name string) Client {
	uuidQueue, err := uuid.NewV4()
	if err != nil {
		log.Criticalf("[Client] Failed to create UUID for reply queue")
		os.Exit(1)
	}
	return &RabbitClient{
		Name:       name,
		inflight:   newInflightRegistry(),
		connection: rabbit.NewRabbitConnection(),
		replyTo:    fmt.Sprintf("replyTo-%s-%s", name, uuidQueue.String()),
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
	correlation, err := uuid.NewV4()
	if err != nil {
		log.Error("[Client] Failed to create correlationId in client")
		return errors.New("client.call.uuid.error")
	}

	replyChannel := c.inflight.push(correlation.String())
	defer close(replyChannel)

	requestBody, err := proto.Marshal(req)
	if err != nil {
		log.Error("[Client] Failed to marshal request")
		return errors.New("client.call.marshal.error")
	}

	message := amqp.Publishing{
		CorrelationId: correlation.String(),
		Timestamp:     time.Now().UTC(),
		Body:          requestBody,
		ReplyTo:       c.replyTo,
	}

	err = c.connection.Publish(rabbit.Exchange, routingKey, message)
	if err != nil {
		log.Errorf("[Client] Failed to publish to %s", routingKey)
		return fmt.Errorf("client.call.publish.%s.error", routingKey)
	}

	select {
	case delivery := <-replyChannel:
		if err := proto.Unmarshal(delivery.Body, res); err != nil {
			return fmt.Errorf("client.unmarshal.%s-reply.error", routingKey)
		}
		return nil
	case <-time.After(1 * time.Second):
		log.Criticalf("[Client] Client timeout on delivery")
		return fmt.Errorf("client.call.timeout.%s.error", routingKey)
	}
}