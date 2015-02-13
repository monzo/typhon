package transport

type Endpoint interface {
	Name() string
	HandleRequest(Request) ([]byte, error)
}
