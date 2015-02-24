package errors

// Error represents all errors which can be passed between services
type Error interface {
	Code() string
	Error() string
	Type() ErrorType
}

// ErrorType is an enumerated type of error
type ErrorType string

const (
	// ErrUnknown indicates an unknown type of error
	// @todo should this just be mapped to an internal server error?
	ErrUnknown = ErrorType("UnknownError")

	ErrBadRequest     = ErrorType("BadRequestError")
	ErrBadResponse    = ErrorType("BadResponseError")
	ErrForbidden      = ErrorType("ForbiddenError")
	ErrInternalServer = ErrorType("InternalServerError")
	ErrNotFound       = ErrorType("NotFoundError")
	ErrTimeout        = ErrorType("TimeoutError")
)

// platformError implements the Error interface, and is the internal type we
// use to pass errors between services. The error cannot be directly instantiated,
// and one of the helper methods should be used to construct a specific type of error
type platformError struct {
	errorType   ErrorType
	code        string
	description string
}

// Code defines a clearly defined inter-service error code
func (p *platformError) Code() string {
	if p != nil {
		return p.code
	}

	return ""
}

// Error returns a string description of the error
func (p *platformError) Error() string {
	if p != nil {
		return p.description
	}

	return ""
}

// Type of error that this error represents
func (p *platformError) Type() ErrorType {
	if p != nil {
		return p.errorType
	}

	return ErrUnknown
}

// InternalServerError creates a new error that represents an error originating within a service
func InternalServerError(code, description string, context ...string) Error {
	return &platformError{
		errorType:   ErrInternalServer,
		code:        code,
		description: description,
	}
}

// BadRequest creates a new error to represent an error caused by the client sending
// an invalid request. This is non-retryable unless the request is modified.
func BadRequest(code, description string, context ...string) Error {
	return &platformError{
		errorType:   ErrBadRequest,
		code:        code,
		description: description,
	}
}

// BadResponse creates a new error representing a failure to response with a valid response
// Examples of this would be a handler returning an invalid message format
func BadResponse(code, description string, context ...string) Error {
	return &platformError{
		errorType:   ErrBadResponse,
		code:        code,
		description: description,
	}
}

// Timeout creates a new error representing a timeout from client to server
func Timeout(code, description string, context ...string) Error {
	return &platformError{
		errorType:   ErrTimeout,
		code:        code,
		description: description,
	}
}

// NotFound creates a new error representing a resource that cannot be found. In some
// cases this is not an error, and would be better represented by a zero length slice of elements
func NotFound(code, description string, context ...string) Error {
	return &platformError{
		errorType:   ErrNotFound,
		code:        code,
		description: description,
	}
}

// Forbidden creates a new error representing a resource that cannot be accessed with
// the current authorisation credentials. The user may need authorising, or if authorised,
// may not be permitted to perform this action
func Forbidden(code, description string, context ...string) Error {
	return &platformError{
		errorType:   ErrForbidden,
		code:        code,
		description: description,
	}
}
