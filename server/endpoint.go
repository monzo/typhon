package server

type Endpoint interface {
	Name() string
	HandleRequest(req Request) (Response, error)
}

type DefaultEndpoint struct {
	EndpointName string
	Handler      func(req Request) (Response, error)
}

func (d *DefaultEndpoint) Name() string {
	return d.EndpointName
}

func (d *DefaultEndpoint) HandleRequest(req Request) (Response, error) {
	return d.Handler(req)
}
