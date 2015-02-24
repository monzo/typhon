package server

import (
	"fmt"
	"reflect"

	"github.com/golang/protobuf/proto"
)

type Endpoint interface {
	Name() string
	HandleRequest(req Request) (Response, error)

	// RequestType returns a pointer to an instance of the expected response type
	RequestType() interface{}

	// ResponseType returns a pointer to an instance of the expected response type
	ResponseType() interface{}
}

type HandlerFunc func(Request) (Response, error)

// @todo DefaultEndpoint is a bit of a misnomer
type DefaultEndpoint struct {
	EndpointName string
	Handler      func(req Request) (Response, error)
	Request      interface{}
	Response     interface{}
}

func (e *DefaultEndpoint) RequestType() interface{} {
	return e.Request
}

func (e *DefaultEndpoint) ResponseType() interface{} {
	return e.Response
}

func (e *DefaultEndpoint) Name() string {
	return e.EndpointName
}

func (e *DefaultEndpoint) HandleRequest(req Request) (Response, error) {

	// TODO check that `Request` and `Response` are set in RegisterEndpoint
	// TODO don't tightly couple `HandleRequest` to the proto encoding

	if e.RequestType() != nil {
		body := cloneTypedPtr(e.RequestType()).(proto.Message)
		if err := proto.Unmarshal(req.Payload(), body); err != nil {
			return nil, fmt.Errorf("Count not unmarshal request")
		}
		req.SetBody(body)
	}

	return e.Handler(req)

	// TODO return error if e.ResponseType() is set and doesn't match
}

// cloneTypedPtr takes a pointer of any type and returns a pointer to
// to a newly allocated instance of that same type.
// This allows us to write generic unmarshalling code that is independent
// of a endpoint's expected message type. This way we can handle
// unmarshalling errors outside of the actual endpoint handler methods.
func cloneTypedPtr(reqType interface{}) interface{} {
	// http://play.golang.org/p/MJOc3g7t23
	// `reflect.New` gives us a `reflect.Value`, using type of `reflect.TypeIf(e.RequestType()).Elem()` (a struct type)
	// and that struct's zero value for a value.
	// `reflectValue.Interface()` puts the type and value back together into an interface type
	reflectValue := reflect.New(reflect.TypeOf(reqType).Elem())
	return reflectValue.Interface()
}
