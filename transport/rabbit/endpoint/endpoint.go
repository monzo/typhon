package endpoint

import "github.com/golang/protobuf/proto"

type Endpoint interface {
	Name() string
	HandleRequest(req *RabbitRequest) (proto.Message, error)
}
