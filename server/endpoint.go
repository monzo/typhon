package server

import (
	"fmt"
	"reflect"

	log "github.com/cihub/seelog"
	"github.com/golang/protobuf/proto"
)

type Endpoint struct {
	Name     string
	Handler  func(Request) (Response, error)
	Request  interface{}
	Response interface{}
}

func (e *Endpoint) HandleRequest(req Request) (Response, error) {

	// TODO check that `Request` and `Response` are set in RegisterEndpoint
	// TODO don't tightly couple `HandleRequest` to the proto encoding

	if e.Request != nil {
		body := cloneTypedPtr(e.Request).(proto.Message)
		if err := proto.Unmarshal(req.Payload(), body); err != nil {
			return nil, fmt.Errorf("Could not unmarshal request")
		}
		req.SetBody(body)
	}

	log.Debugf("%s.%s handler received request: %+v", req.Service(), e.Name, req.Body())

	resp, err := e.Handler(req)

	if err != nil {
		log.Errorf("%s.%s handler error: %s", req.Service(), e.Name, err.Error())
	} else {
		log.Debugf("%s.%s handler response: %+v", req.Service(), e.Name, resp.(*ProtoResponse).Pb)
	}

	return resp, err
	// TODO return error if e.Response is set and doesn't match
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
