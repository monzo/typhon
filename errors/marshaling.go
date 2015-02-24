package errors

import (
	pe "github.com/b2aio/typhon/proto/error"
)

// Marshal an error into a protobuf for transmission
func Marshal(e *platformError) *pe.PlatformError {

	// Account for nil errors
	if e == nil {
		return &pe.PlatformError{
			Type:        pe.ErrorType_UNKNOWN,
			Code:        "unknown",
			Description: "Unknown error, nil error marshalled",
		}
	}

	return &pe.PlatformError{
		Type:        errorTypeToProto(e.Type()),
		Code:        e.Code(),
		Description: e.Description(),
	}
}

// Unmarshal a protobuf error into a local error
func Unmarshal(p *pe.PlatformError) *platformError {
	if p == nil {
		return &platformError{}
	}

	return &platformError{
		errorType:   protoToErrorType(p.Type),
		code:        p.Code,
		description: p.Description,
	}
}

// protoToErrorType marshals a protobuf error type to a local error type
func protoToErrorType(p pe.ErrorType) ErrorType {
	if e, ok := pe.ErrorType_name[int32(p)]; ok {
		return ErrorType(e)
	}
	return ErrUnknown
}

// protoToErrorType marshals a protobuf error type to a local error type
func errorTypeToProto(e ErrorType) pe.ErrorType {
	if p, ok := pe.ErrorType_value[string(e)]; ok {
		return pe.ErrorType(p)
	}

	return pe.ErrorType_UNKNOWN
}
