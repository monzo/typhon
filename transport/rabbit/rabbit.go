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
	Exchange = os.Getenv("RABBIT_EXCHANGE")
}

func NewConnection() chan *RabbitConnection {
	conn := &RabbitConnection{}
	result := make(chan *RabbitConnection, 1)
	go conn.Connect(result)
	return result
}

type RabbitConnection struct {
	Connection     *amqp.Connection
	Channel        *RabbitChannel
	DefaultChannel *RabbitChannel
}

func (r *RabbitConnection) Connect(connected chan *RabbitConnection) {
	for {
		if err := r.tryToConnect(); err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		connected <- r
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
	r.Channel.DeclareExchange(Exchange)
}
