package rabbit

import (
	"os"
	"time"

	log "github.com/cihub/seelog"
	"github.com/nu7hatch/gouuid"
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
	if Exchange == "" {
		log.Criticalf("RABBIT_EXCHANGE is required")
		os.Exit(1)
	}
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
	if err != nil {
		log.Error("[Rabbit] Failed to create Bunny Channel")
		return err
	}
	r.Channel.DeclareExchange(Exchange)
	r.DefaultChannel, err = NewRabbitChannel(r.Connection)
	if err != nil {
		log.Error("[Rabbit] Failed to create default Channel")
		return err
	}
	log.Info("[Rabbit] Connected to RabbitMQ")
	return nil
}

func (r *RabbitTransport) Consume(serverName string) <-chan transport.Request {
	consumerChannel, err := NewRabbitChannel(r.Connection)
	if err != nil {
		log.Errorf("[Rabbit] Failed to create new channel")
		log.Error(err.Error())
	}
	err = consumerChannel.DeclareQueue(serverName)
	if err != nil {
		log.Errorf("[Rabbit] Failed to declare queue")
		log.Error(err.Error())
	}
	consumer := make(chan transport.Request)
	err = consumerChannel.BindQueue(serverName, Exchange)
	if err != nil {
		log.Errorf("[Rabbit] Failed to bind queue to exchange")
		log.Error(err.Error())
	}
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

func (r *RabbitTransport) PublishFromRequest(request transport.Request, body []byte, err error) {
	rabbitRequest := request.(*RabbitRequest)
	msg := amqp.Publishing{
		CorrelationId: rabbitRequest.CorrelationID(),
		Timestamp:     time.Now().UTC(),
		Body:          body,
	}
	r.DefaultChannel.Publish("", rabbitRequest.ReplyTo(), msg)
}

func (r *RabbitTransport) Publish(routingKey string, body []byte) {
	correlationID, _ := uuid.NewV4()
	msg := amqp.Publishing{
		CorrelationId: correlationID.String(),
		Timestamp:     time.Now().UTC(),
		Body:          body,
		ReplyTo:       "aaa",
	}
	log.Infof("[Rabbit] Publishing message to %s", routingKey)
	err := r.DefaultChannel.Publish(Exchange, routingKey, msg)
	if err != nil {
		log.Error(err.Error())
	}
}
