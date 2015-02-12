package transport

import "reflect"

type Request struct {
	delivery reflect.Value
}

func NewRequest(delivery interface{}) *Request {
	return &Request{
		delivery: reflect.ValueOf(delivery),
	}
}
