package endpoint

import (
	"errors"

	"github.com/golang/protobuf/proto"
	"github.com/vinceprignano/bunny/transport"
)

type JsonEndpoint struct {
	EndpointName string
	Transport    transport.Transport
	Handler      func(delivery *transport.Request) (proto.Message, error)
}

func (j *JsonEndpoint) Name() string {
	return j.EndpointName
}

func (j *JsonEndpoint) HandleRequest(req *transport.Request) ([]byte, error) {
	res, err := j.Handler(req)
	if err != nil {
		return nil, errors.New("Failed")
	}
	return proto.Marshal(res)
}
