package rabbit

import (
	"os"
	"time"

	log "github.com/cihub/seelog"
	"github.com/streadway/amqp"
	"github.com/vinceprignano/bunny/transport"
)

var (
	RabbitURL string
	Exchange  string
)

func init() {
	RabbitURL = os.Getenv("RABBIT_URL")
	Exchange = os.Getenv("RABBIT_EXCHANGE")
}

func NewRabbitTransport() *RabbitTransport {
	return &RabbitTransport{
		notify: make(chan bool, 1),
	}
}

type RabbitTransport struct {
	Connection     *amqp.Connection
	Channel        *RabbitChannel
	DefaultChannel *RabbitChannel
	notify         chan bool
}

func (r *RabbitTransport) Init() chan bool {
	go r.Connect(r.notify)
	return r.notify
}

func (r *RabbitTransport) Connect(connected chan bool) {
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

func (r *RabbitTransport) tryToConnect() error {
	var err error
	r.Connection, err = amqp.Dial(RabbitURL)
	if err != nil {
		log.Error("[Rabbit] Failed to establish connection with RabbitMQ")
		return err
	}
	r.Channel, err = NewRabbitChannel(r.Connection)
	r.Channel.DeclareExchange(Exchange)

	return nil
}

func (r *RabbitTransport) Consume(serverName string) <-chan transport.Request {
	consumerChannel, err := NewRabbitChannel(r.Connection)
	consumerChannel.DeclareQueue(serverName)
	consumer := make(chan transport.Request)
	consumerChannel.BindQueue(serverName, Exchange)
	messages, err := consumerChannel.ConsumeQueue(serverName)
	if err != nil {
		close(consumer)
		log.Errorf("[Rabbit] Failed to consume from %s queue", serverName)
		log.Error(err.Error())
	}
	go func() {
		for msg := range messages {
			consumer <- NewRabbitRequest(&msg)
		}
	}()
	return consumer
}
