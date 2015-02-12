package endpoint

import (
	"errors"

	"github.com/golang/protobuf/proto"
	"github.com/vinceprignano/bunny/transport"
)

type ProtoEndpoint struct {
	EndpointName string
	Transport    transport.Transport
	Handler      func(delivery *transport.Request) (proto.Message, error)
}

func (p *ProtoEndpoint) Name() string {
	return p.EndpointName
}

func (p *ProtoEndpoint) HandleRequest(req *transport.Request) ([]byte, error) {
	res, err := p.Handler(req)
	if err != nil {
		return nil, errors.New("Failed")
	}
	return proto.Marshal(res)
}
