package errors

import (
	pe "github.com/b2aio/typhon/proto/error"
)

// Marshal an error into a protobuf for transmission
func Marshal(e *ServiceError) *pe.Error {

	// Account for nil errors
	if e == nil {
		return &pe.Error{
			Type:        pe.ErrorType_UNKNOWN,
			Code:        "unknown",
			Description: "Unknown error, nil error marshalled",
		}
	}

	return &pe.Error{
		Type:           errorTypeToProto(e.Type()),
		Code:           e.Code(),
		Description:    e.Description(),
		ClientCode:     int32(e.ClientCode()),
		PublicContext:  e.PublicContext(),
		PrivateContext: e.PrivateContext(),
	}
}

// Unmarshal a protobuf error into a local error
func Unmarshal(p *pe.Error) *ServiceError {
	if p == nil {
		// @todo should this actually be blank?
		// or should we put a code and description in, like on marshaling
		return &ServiceError{}
	}

	return &ServiceError{
		errorType:      protoToErrorType(p.Type),
		code:           p.Code,
		description:    p.Description,
		clientCode:     int(p.ClientCode),
		publicContext:  p.PublicContext,
		privateContext: p.PrivateContext,
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
