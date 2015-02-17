package rabbit

import (
	"os"
	"time"

	log "github.com/cihub/seelog"
	"github.com/streadway/amqp"
)

var (
	RabbitURL string
	Exchange  string
)

func init() {
	RabbitURL = os.Getenv("RABBIT_URL")
	if RabbitURL == "" {
		RabbitURL = "amqp://localhost:5672"
		log.Infof("Setting RABBIT_URL to default value %s", RabbitURL)
	}
	Exchange = os.Getenv("RABBIT_EXCHANGE")
	if Exchange == "" {
		Exchange = "amq.topic"
		log.Infof("Setting RABBIT_EXCHANGE to default value %s", Exchange)
	}
}

func NewRabbitConnection() *RabbitConnection {
	return &RabbitConnection{
		notify: make(chan bool, 1),
	}
}

type RabbitConnection struct {
	Connection      *amqp.Connection
	Channel         *RabbitChannel
	ExchangeChannel *RabbitChannel
	notify          chan bool
}

func (r *RabbitConnection) Init() chan bool {
	go r.Connect(r.notify)
	return r.notify
}

func (r *RabbitConnection) Connect(connected chan bool) {
	for {
		if err := r.tryToConnect(); err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		connected <- true
		notifyClose := make(chan *amqp.Error)
		r.Connection.NotifyClose(notifyClose)
		<-notifyClose
	}
}

func (r *RabbitConnection) tryToConnect() error {
	var err error
	r.Connection, err = amqp.Dial(RabbitURL)
	if err != nil {
		log.Error("[Rabbit] Failed to establish connection with RabbitMQ")
		return err
	}
	r.Channel, err = NewRabbitChannel(r.Connection)
	if err != nil {
		log.Error("[Rabbit] Failed to create Bunny Channel")
		return err
	}
	r.Channel.DeclareExchange(Exchange)
	r.ExchangeChannel, err = NewRabbitChannel(r.Connection)
	if err != nil {
		log.Error("[Rabbit] Failed to create default Channel")
		return err
	}
	log.Info("[Rabbit] Connected to RabbitMQ")
	return nil
}

func (r *RabbitConnection) Consume(serverName string) (<-chan amqp.Delivery, error) {
	consumerChannel, err := NewRabbitChannel(r.Connection)
	if err != nil {
		log.Errorf("[Rabbit] Failed to create new channel")
		log.Error(err.Error())
	}
	err = consumerChannel.DeclareQueue(serverName)
	if err != nil {
		log.Errorf("[Rabbit] Failed to declare queue %s", serverName)
		log.Error(err.Error())
	}
	err = consumerChannel.BindQueue(serverName, Exchange)
	if err != nil {
		log.Errorf("[Rabbit] Failed to bind %s to %s exchange", serverName, Exchange)
	}
	return consumerChannel.ConsumeQueue(serverName)
}

func (r *RabbitConnection) Publish(exchange, routingKey string, msg amqp.Publishing) error {
	return r.ExchangeChannel.Publish(exchange, routingKey, msg)
}
