package transport

type Endpoint interface {
	Name() string
	HandleRequest(req Request) ([]byte, error)
}
