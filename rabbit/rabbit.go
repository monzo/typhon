package rabbit

import (
	"os"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/streadway/amqp"
)

var (
	RabbitURL string
	Exchange  string
)

func NewRabbitConnection() *RabbitConnection {

	// Initialize rabbitmq connection from env vars. The defaults are expected to work
	// in development. ci, dev-ci and mesos override these defaults
	// @todo Refactor initialization of the connection to allow the "server" abstraction we'll introduce to inject the configuration
	// @todo once config server has http interface, we should use values from the config
	// server to connect to rabbit
	RabbitURL = os.Getenv("RABBIT_URL")
	if RabbitURL == "" {
		RabbitURL = "amqp://admin:guest@192.168.59.103:5672"
		log.Infof("Setting RABBIT_URL to default value %s", RabbitURL)
	}
	log.Infof("Set RABBIT_URL to %s", RabbitURL)
	Exchange = os.Getenv("RABBIT_EXCHANGE")
	if Exchange == "" {
		Exchange = "b2a"
		log.Infof("Setting RABBIT_EXCHANGE to default value %s", Exchange)
	}
	log.Infof("Set RABBIT_EXCHANGE to %s", Exchange)

	return &RabbitConnection{
		notify:    make(chan bool, 1),
		closeChan: make(chan struct{}),
	}
}

type RabbitConnection struct {
	Connection      *amqp.Connection
	Channel         *RabbitChannel
	ExchangeChannel *RabbitChannel
	notify          chan bool

	connected bool

	mtx       sync.Mutex
	closeChan chan struct{}
	closed    bool
}

func (r *RabbitConnection) Init() chan bool {
	go r.Connect(r.notify)
	return r.notify
}

func (r *RabbitConnection) Connect(connected chan bool) {
	for {
		log.Debugf("[Rabbit] Attempting to connect")
		if err := r.tryToConnect(); err != nil {
			log.Debugf("[Rabbit] Failed to connect, sleeping 1s")
			time.Sleep(1 * time.Second)
			continue
		}
		connected <- true
		r.connected = true
		notifyClose := make(chan *amqp.Error)
		r.Connection.NotifyClose(notifyClose)

		// Block until we get disconnected, or shut down
		select {
		case err := <-notifyClose:
			// Spin around and reconnect
			r.connected = false
			log.Debugf("[Rabbit] AMQP connection closed (notifyClose): %s", err.Error())
		case <-r.closeChan:
			// Shut down connection
			log.Debugf("[Rabbit] Closing AMQP connection (closeChan closed)")
			if err := r.Connection.Close(); err != nil {
				log.Errorf("Failed to close AMQP connection: %v", err)
			}
			r.connected = false
			return
		}
	}
}

func (r *RabbitConnection) IsConnected() bool {
	return r.connected
}

func (r *RabbitConnection) Close() {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	if r.closed {
		return
	}

	close(r.closeChan)
	r.closed = true
}

func (r *RabbitConnection) tryToConnect() error {
	var err error
	r.Connection, err = amqp.Dial(RabbitURL)
	if err != nil {
		log.Errorf("[Rabbit] Failed to establish connection with RabbitMQ: %s", RabbitURL)
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
		log.Errorf("[Rabbit] Failed to create new channel: %s", err.Error())
		return nil, err
	}

	err = consumerChannel.DeclareQueue(serverName)
	if err != nil {
		log.Errorf("[Rabbit] Failed to declare queue %s: %s", serverName, err.Error())
		return nil, err
	}

	deliveries, err := consumerChannel.ConsumeQueue(serverName)
	if err != nil {
		log.Errorf("[Rabbit] Failed to declare queue %s: %s", serverName, err.Error())
		return nil, err
	}

	err = consumerChannel.BindQueue(serverName, Exchange)
	if err != nil {
		log.Errorf("[Rabbit] Failed to bind %s to %s exchange: %s", serverName, Exchange, err.Error())
		return nil, err
	}

	return deliveries, nil
}

func (r *RabbitConnection) Publish(exchange, routingKey string, msg amqp.Publishing) error {
	return r.ExchangeChannel.Publish(exchange, routingKey, msg)
}
