package errors

import (
	pe "github.com/b2aio/typhon/proto/error"
)

// Marshal an error into a protobuf for transmission
func Marshal(e *Error) *pe.Error {

	// Account for nil errors
	if e == nil {
		return &pe.Error{
			Code:    ErrUnknown,
			Message: "Unknown error, nil error marshalled",
		}
	}

	return &pe.Error{
		Code:           int32(e.Code),
		Message:        e.Message,
		PublicContext:  e.PublicContext,
		PrivateContext: e.PrivateContext,
	}
}

// Unmarshal a protobuf error into a local error
func Unmarshal(p *pe.Error) *Error {
	if p == nil {
		return &Error{
			Message: "Nil error unmarshalled!",
		}
	}

	return &Error{
		Code:           int(p.Code),
		Message:        p.Message,
		PublicContext:  p.PublicContext,
		PrivateContext: p.PrivateContext,
	}
}
