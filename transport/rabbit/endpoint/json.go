package endpoint

type JsonEndpoint struct {
	EndpointName string
	Handler      func(interface{}) ([]byte, error)
}

func (j *JsonEndpoint) Name() string {
	return j.EndpointName
}

func (j *JsonEndpoint) HandleRequest(delivery interface{}) ([]byte, error) {
	return j.Handler(delivery)
}
