package endpoint

type ProtoEndpoint struct {
	EndpointName string
	Handler      func(delivery interface{}) ([]byte, error)
}

func (p *ProtoEndpoint) Name() string {
	return p.EndpointName
}

func (p *ProtoEndpoint) HandleRequest(delivery interface{}) ([]byte, error) {
	return p.Handler(delivery)
}
