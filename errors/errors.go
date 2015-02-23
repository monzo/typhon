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
	UnknownError        = ErrorType("UnknownError")
	InternalServerError = ErrorType("InternalServerError")
	BadRequestError     = ErrorType("BadRequestError")
	BadResponseError    = ErrorType("BadResponseError")
	TimeoutError        = ErrorType("TimeoutError")
	NotFoundError       = ErrorType("NotFoundError")
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

	return UnknownError
}

// NewInternalServerError creates a new error that represents an error originating within a service
func NewInternalServerError(code, description string, context ...string) Error {
	return &platformError{
		errorType:   InternalServerError,
		code:        code,
		description: description,
	}
}

// NewBadRequestError creates a new error to represent an error caused by the client sending
// an invalid request. This is non-retryable unless the request is modified.
func NewBadRequestError(code, description string, context ...string) Error {
	return &platformError{
		errorType:   BadRequestError,
		code:        code,
		description: description,
	}
}

// NewBadResponseError creates a new error representing a failure to response with a valid response
// Examples of this would be a handler returning an invalid message format
func NewBadResponseError(code, description string, context ...string) Error {
	return &platformError{
		errorType:   BadResponseError,
		code:        code,
		description: description,
	}
}

// NewTimeoutError creates a new error representing a timeout from client to server
func NewTimeoutError(code, description string, context ...string) Error {
	return &platformError{
		errorType:   TimeoutError,
		code:        code,
		description: description,
	}
}

// NewNotFoundError creates a new error representing a resource that cannot be found. In some
// cases this is not an error, and would be better represented by a zero length slice of elements
func NewNotFoundError(code, description string, context ...string) Error {
	return &platformError{
		errorType:   NotFoundError,
		code:        code,
		description: description,
	}
}
