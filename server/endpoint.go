package server

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/b2aio/typhon/auth"
	"github.com/b2aio/typhon/errors"
	log "github.com/cihub/seelog"
	"github.com/golang/protobuf/proto"
)

type Endpoint struct {
	Name       string
	Handler    func(Request) (proto.Message, error)
	Request    interface{}
	Response   interface{}
	Authorizer auth.Authorizer

	// server is a reference to the parent server this endpoint is registered with
	server Server
}

func (e *Endpoint) HandleRequest(req Request) (proto.Message, error) {
	var err error

	// @todo check that `Request` and `Response` are set in RegisterEndpoint

	if e.Request != nil {
		body := cloneTypedPtr(e.Request).(proto.Message)
		if err := unmarshalRequest(req, body); err != nil {
			return nil, err
		}
		req.SetBody(body)
	}

	log.Debugf("[Server] %s.%s handler received request: %+v", req.Service(), e.Name, req.Body())

	// Authenticate access to this endpoint
	if err := authenticateEndpointAccess(e, req); err != nil {
		log.Warnf("Failed to authenticate access to %s endpoint", e.Name)
		return nil, err
	}

	resp, err := e.Handler(req)

	if err != nil {
		err = enrichError(err, req, e)
		log.Errorf("[Server] %s.%s handler error: %s", req.Service(), e.Name, err.Error())
	} else {
		log.Debugf("[Server] %s.%s handler response: %+v", req.Service(), e.Name, resp)
	}

	return resp, err
	// @todo return error if e.Response is set and doesn't match
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

// enrichError converts an error interface into *errors.Error and attaches
// lots of information to it.
// NOTE: if the error came from somewhere down the stack, it isn't modified
// @todo once the server context gives us a parent request and trace id, we can store even more information in the error!
func enrichError(err error, ctx Request, endpoint *Endpoint) *errors.Error {
	wrappedErr := errors.Wrap(err)

	// @todo perhaps make methods PrivateContext() and PublicContext() that
	// to deal with nil contexts
	if wrappedErr.PrivateContext == nil {
		wrappedErr.PrivateContext = map[string]string{}
	}

	// @todo an error will probably have a source_request_id or something that we can use to
	// more reliably make sure this information is only attached once, as the error travels up the service stack
	if wrappedErr.PrivateContext["service"] == "" {
		wrappedErr.PrivateContext["service"] = ctx.Service()
	}
	if wrappedErr.PrivateContext["endpoint"] == "" {
		wrappedErr.PrivateContext["endpoint"] = endpoint.Name
	}
	return wrappedErr
}

// unmarshalRequest payload into the body based on content-type
func unmarshalRequest(req Request, body proto.Message) (err error) {
	if len(req.Payload()) == 0 {
		return nil
	}

	if req.ContentType() == "application/x-protobuf" {
		return proto.Unmarshal(req.Payload(), body)
	} else {
		return json.Unmarshal(req.Payload(), body)
	}

	return errors.Wrap(fmt.Errorf("Could not unmarshal request"))
}
