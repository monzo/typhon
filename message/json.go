package message

import (
	"encoding/json"
	"reflect"
)

const JSONContentType = "application/json"

type jsonMarshaler struct{}

func (j *jsonMarshaler) MarshalBody(msg Message) error {
	if body := msg.Body(); body != nil {
		payload, err := json.Marshal(body)
		if err == nil {
			msg.SetPayload(payload)
			msg.SetHeader("Content-Type", JSONContentType)
		}
		return err
	} else {
		return nil
	}
}

// It's safe to share a common JSON marshaler, so we return a singleton.
var sharedJSONMarshaler Marshaler = &jsonMarshaler{}

// JSONMarshaler returns a Marshaler that marshals a struct into JSON.
func JSONMarshaler() Marshaler {
	return sharedJSONMarshaler
}

type jsonUnmarshaler struct {
	T reflect.Type
}

func (ju *jsonUnmarshaler) UnmarshalPayload(msg Message) error {
	var result interface{}

	_body := msg.Body()
	if bodyT := reflect.TypeOf(_body); bodyT != nil && bodyT.AssignableTo(pu.T) {
		// The message already has an appropriate body; unmarshal into it
		result = _body
	} else {
		// No body (or an inappropriate type); overwrite with a new object
		result = reflect.New(ju.T.Elem()).Interface()
	}

	err := json.Unmarshal(msg.Payload(), result)
	if err == nil {
		msg.SetBody(result)
	}
	return err
}

// JSONUnmarshaler returns an Unmarshaler that unmarshals JSON into a decoded Body.
// A "template" object must be provided (an object of the appropriate type).
func JSONUnmarshaler(protocol interface{}) Unmarshaler {
	return &jsonUnmarshaler{
		T: reflect.TypeOf(protocol),
	}
}
