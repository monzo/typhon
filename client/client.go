package client

import (
	"errors"
	"fmt"
	"os"
	"time"

	log "github.com/cihub/seelog"
	"github.com/golang/protobuf/proto"
	"github.com/nu7hatch/gouuid"
	"github.com/streadway/amqp"
	"github.com/vinceprignano/bunny/rabbit"
)

var Exchange string

type Client struct {
	Name       string
	dispatcher *dispatcher
	replyTo    string
	connection *rabbit.RabbitConnection
}

func init() {
	Exchange = os.Getenv("RABBIT_EXCHANGE")
}

var NewClient = func(name string) *Client {
	uuidQueue, err := uuid.NewV4()
	if err != nil {
		log.Criticalf("[Client] Failed to create UUID for reply queue")
		os.Exit(1)
	}
	return &Client{
		Name:       name,
		dispatcher: newDispatcher(),
		connection: rabbit.NewRabbitConnection(),
		replyTo:    fmt.Sprintf("replyTo-%s-%s", name, uuidQueue.String()),
	}
}

func (c *Client) Init() {
	select {
	case <-c.connection.Init():
		log.Info("[Client] Connected to transport")
	case <-time.After(10 * time.Second):
		log.Critical("[Client] Failed to connect to transport")
		os.Exit(1)
	}
	c.initConsume()
}

func (c *Client) initConsume() {
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

func (c *Client) handleDelivery(delivery amqp.Delivery) {
	channel := c.dispatcher.pop(delivery.CorrelationId)
	if channel == nil {
		log.Errorf("[Client] CorrelationID -> %s does not exist in dispatcher", delivery.CorrelationId)
		return
	}
	select {
	case channel <- delivery:
	default:
		log.Errorf("[Client] Error in delivery for correlation %s", delivery.CorrelationId)
	}
}

func (c *Client) Call(routingKey string, req proto.Message, res proto.Message) error {
	correlation, err := uuid.NewV4()
	if err != nil {
		log.Error("[Client] Failed to create correlationId in client")
		return errors.New("client.call.uuid.error")
	}

	replyChannel := c.dispatcher.add(correlation.String())
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

	err = c.connection.Publish(Exchange, routingKey, message)
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
	case <-time.After(10 * time.Second):
		log.Criticalf("[Client] Client timeout on delivery")
		return fmt.Errorf("client.call.timeout.%s.error", routingKey)
	}
}
