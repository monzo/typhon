package message

import (
	"encoding/json"
	"errors"
	"reflect"

	"github.com/golang/protobuf/proto"
)

const (
	ProtoContentType = "application/x-protobuf"
)

type protoMarshaler struct{}

func (p *protoMarshaler) MarshalBody(msg Message) error {
	switch body := msg.Body().(type) {
	case proto.Message:
		payload, err := proto.Marshal(body)
		if err == nil {
			msg.SetPayload(payload)
			msg.SetHeader("Content-Type", ProtoContentType)
		}
		return err

	default:
		return errors.New("Protobuf request marshaler can only marshal proto.Message objects")
	}
}

// It's safe to share a common proto marshaler, so we return a singleton.
var sharedProtoMarshaler Marshaler = &protoMarshaler{}

// ProtoMarshaler returns a Marshaler that marshals a protobuf struct into the wire format.
func ProtoMarshaler() Marshaler {
	return sharedProtoMarshaler
}

type protoUnmarshaler struct {
	T reflect.Type
}

func (pu *protoUnmarshaler) UnmarshalPayload(msg Message) error {
	result := proto.Message(nil)
	err := error(nil)

	_body := msg.Body()
	if bodyT := reflect.TypeOf(_body); bodyT != nil && bodyT.AssignableTo(pu.T) {
		// The message already has an appropriate body; unmarshal into it
		result = _body.(proto.Message)
	} else {
		// No body (or an inappropriate type); overwrite with a new object
		result = reflect.New(pu.T.Elem()).Interface().(proto.Message)
	}

	if msg.Headers()["Content-Type"] == ProtoContentType {
		err = proto.Unmarshal(msg.Payload(), result)
	} else {
		err = json.Unmarshal(msg.Payload(), result)
	}

	if err == nil {
		msg.SetBody(result)
	}
	return err
}

// ProtoUnmarshaler returns an Unmarshaler that unmarshals wire-format protobuf (or JSON protobuf) into a decoded Body.
// A "template" object must be provided (an object of the appropriate type).
func ProtoUnmarshaler(protocol proto.Message) Unmarshaler {
	return &protoUnmarshaler{
		T: reflect.TypeOf(protocol),
	}
}
