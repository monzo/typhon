package rabbit

import (
	"errors"
	"fmt"

	"github.com/nu7hatch/gouuid"
	"github.com/streadway/amqp"
)

type RabbitChannel struct {
	uuid       string
	connection *amqp.Connection
	channel    *amqp.Channel
}

func NewRabbitChannel(conn *amqp.Connection) (*RabbitChannel, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	rabbitCh := &RabbitChannel{
		uuid:       id.String(),
		connection: conn,
	}
	if err := rabbitCh.Connect(); err != nil {
		return nil, err
	}
	return rabbitCh, nil

}

func (r *RabbitChannel) Connect() error {
	var err error
	r.channel, err = r.connection.Channel()
	if err != nil {
		return err
	}
	return nil
}

func (r *RabbitChannel) Close() error {
	if r.channel == nil {
		return errors.New("Channel is nil")
	}
	return r.channel.Close()
}

func (r *RabbitChannel) Publish(exchange string, routingKey string, message amqp.Publishing) error {
	if r.channel == nil {
		return errors.New("Channel is nil")
	}
	return r.channel.Publish(exchange, routingKey, false, false, message)
}

func (r *RabbitChannel) DeclareExchange(exchange string) error {
	return r.channel.ExchangeDeclare(exchange, "topic", false, false, false, false, nil)
}

func (r *RabbitChannel) DeclareQueue(queue string) error {
	_, err := r.channel.QueueDeclare(queue, false, false, false, false, nil)
	return err
}

func (r *RabbitChannel) DeclareDurableQueue(queue string) error {
	_, err := r.channel.QueueDeclare(queue, true, false, false, false, nil)
	return err
}

func (r *RabbitChannel) ConsumeQueue(queue string) (<-chan amqp.Delivery, error) {
	return r.channel.Consume(queue, r.uuid, false, false, false, false, nil)
}

func (r *RabbitChannel) BindQueue(queue, exchange string) error {
	return r.channel.QueueBind(queue, fmt.Sprintf("%s.#", queue), exchange, false, nil)
}
