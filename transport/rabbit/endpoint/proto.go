package endpoint

import "github.com/vinceprignano/bunny/transport/rabbit"

type ProtoEndpoint struct {
	EndpointName string
	Handler      func(*rabbit.RabbitRequest) ([]byte, error)
}

func (p *ProtoEndpoint) Name() string {
	return p.EndpointName
}

func (p *ProtoEndpoint) HandleRequest(req *rabbit.RabbitRequest) ([]byte, error) {
	return p.Handler(req)
}
