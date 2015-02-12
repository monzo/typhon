package endpoint

import "github.com/vinceprignano/bunny/transport"

type Endpoint interface {
	Name() string
	HandleRequest(*transport.Request) ([]byte, error)
}
