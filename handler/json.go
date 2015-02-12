package handler

import (
	"encoding/json"

	"github.com/streadway/amqp"
)

type JsonHandler struct{}

func (j *JsonHandler) HandleDelivery(delivery amqp.Delivery) {

}

func (j *JsonHandler) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (j *JsonHandler) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
