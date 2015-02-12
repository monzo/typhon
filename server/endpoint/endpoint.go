package endpoint

type Endpoint interface {
	Name() string
	HandleRequest(interface{}) ([]byte, error)
}
