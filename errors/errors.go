package errors

import (
	"fmt"

	"github.com/mondough/typhon/errors/stack"
)

type Error struct {
	Code        string            `json:"code"`
	Message     string            `json:"message"`
	Params      map[string]string `json:"params"`
	StackFrames stack.Stack       `json:"stack"`
}

// Generic error codes. Each of these has their own constructor for convenience.
// You can use any string as a code, just use the `New` method.
const (
	ErrUnknown         = "err_unknown"
	ErrInternalService = "err_internal_service"
	ErrBadRequest      = "err_bad_request"
	ErrBadResponse     = "err_bad_response"
	ErrForbidden       = "err_forbidden"
	ErrUnauthorized    = "err_unauthorized"
	ErrNotFound        = "err_not_found"
	ErrTimeout         = "err_timeout"
)

// Error returns a string message of the error
// This means the Error implements the error interface
func (p *Error) Error() string {
	if p == nil {
		return ""
	}
	return p.Message
}

// StackString formats the stack as a beautiful string with newlines
func (p *Error) StackString() string {
	stackStr := ""
	for _, frame := range p.StackFrames {
		stackStr = fmt.Sprintf("%s\n  %s:%d in %s", stackStr, frame.Filename, frame.Line, frame.Method)
	}
	return stackStr
}

// VerboseString returns the error message, stack trace and contexts
func (p *Error) VerboseString() string {
	return fmt.Sprintf("%s\nParams: %+v\n%s", p.Error(), p.Params, p.StackString())
}

// New creates a new error for you. Use this if you want to pass along a custom error code.
// Otherwise use the handy shorthand factories below
func New(code string, message string, params map[string]string) *Error {
	return errorFactory(code, message, params)
}

// Wrap takes any error interface and wraps it into an Error.
// This is useful because an Error contains lots of useful goodies, like the stacktrace of the error.
// NOTE: If `err` is already an `Error` the passed contexts will be ignored
func Wrap(err error, params map[string]string) *Error {
	return WrapWithCode(err, params, ErrInternalService)
}

func WrapWithCode(err error, params map[string]string, code string) *Error {
	if err == nil {
		return nil
	}
	switch err := err.(type) {
	case *Error:
		return err
	default:
		return errorFactory(code, err.Error(), params)
	}
}

// InternalService creates a new error to represent an internal service error.
// Only use internal service error if we know very little about the error. Most
// internal service errors will come from `Wrap`ing a vanilla `error` interface
func InternalService(code, message string, params map[string]string) *Error {
	return errorFactory(fmt.Sprintf("%s.%s", ErrInternalService, code), message, params)
}

// BadRequest creates a new error to represent an error caused by the client sending
// an invalid request. This is non-retryable unless the request is modified.
func BadRequest(code, message string, params map[string]string) *Error {
	return errorFactory(fmt.Sprintf("%s.%s", ErrBadRequest, code), message, params)
}

// BadResponse creates a new error representing a failure to response with a valid response
// Examples of this would be a handler returning an invalid message format
func BadResponse(code, message string, params map[string]string) *Error {
	return errorFactory(fmt.Sprintf("%s.%s", ErrBadResponse, code), message, params)
}

// Timeout creates a new error representing a timeout from client to server
func Timeout(code, message string, params map[string]string) *Error {
	return errorFactory(fmt.Sprintf("%s.%s", ErrTimeout, code), message, params)
}

// NotFound creates a new error representing a resource that cannot be found. In some
// cases this is not an error, and would be better represented by a zero length slice of elements
func NotFound(code, message string, params map[string]string) *Error {
	return errorFactory(fmt.Sprintf("%s.%s", ErrNotFound, code), message, params)
}

// Forbidden creates a new error representing a resource that cannot be accessed with
// the current authorisation credentials. The user may need authorising, or if authorised,
// may not be permitted to perform this action
func Forbidden(code, message string, params map[string]string) *Error {
	return errorFactory(fmt.Sprintf("%s.%s", ErrForbidden, code), message, params)
}

// Unauthorized creates a new error indicating that authentication is required,
// but has either failed or not been provided.
func Unauthorized(code, message string, params map[string]string) *Error {
	return errorFactory(fmt.Sprintf("%s.%s", ErrUnauthorized, code), message, params)
}

// errorConstructor returns a `*Error` with the specified code, message and params.
// Builds a stack based on the current call stack
func errorFactory(code string, message string, params map[string]string) *Error {
	err := &Error{
		Code:    ErrUnknown,
		Message: message,
		Params:  map[string]string{},
	}
	if len(code) > 0 {
		err.Code = code
	}
	if params != nil {
		err.Params = params
	}

	// TODO pass in context.Context

	// Build stack and skip first three lines:
	//  - stack.go BuildStack()
	//  - errors.go errorFactory()
	//  - errors.go public constructor method
	err.StackFrames = stack.BuildStack(3)

	return err
}
