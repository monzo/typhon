package errors

import (
	"fmt"

	"github.com/mondough/typhon/errors/stack"
)

type Error interface {
	Code() string
	Message() string
	Params() map[string]string
	StackFrames() stack.Stack
	Error() string // for compatibility with errors.Error interface
}

type errorImpl struct {
	code        string
	message     string
	params      map[string]string
	stackFrames []stack.Frame
}

func (e errorImpl) Code() string {
	if e.code == "" {
		return ErrUnknown
	}
	return e.code
}

func (e errorImpl) Message() string {
	return e.message
}

func (e errorImpl) Params() map[string]string {
	return e.params
}

func (e errorImpl) StackFrames() stack.Stack {
	return e.stackFrames
}

// Generic error codes. Each of these has their own constructor for convenience.
// You can use any integer as a code, just use the `New` method.
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
func (p *errorImpl) Error() string {
	if p == nil {
		return ""
	}
	return p.message
}

// StackString formats the stack as a beautiful string with newlines
func (p *errorImpl) StackString() string {
	stackStr := ""
	for _, frame := range p.stackFrames {
		stackStr = fmt.Sprintf("%s\n  %s:%d in %s", stackStr, frame.Filename(), frame.Line(), frame.Method())
	}
	return stackStr
}

// VerboseString returns the error message, stack trace and contexts
func (p *errorImpl) VerboseString() string {
	return fmt.Sprintf("%s\nParams: %+v\n%s", p.Error(), p.params, p.StackString())
}

// New creates a new error for you. Use this if you want to pass along a custom error code.
// Otherwise use the handy shorthand factories below
func New(code string, message string, params map[string]string) Error {
	return errorFactory(code, message, params)
}

// Wrap takes any error interface and wraps it into an Error.
// This is useful because an Error contains lots of useful goodies, like the stacktrace of the error.
// NOTE: If `err` is already an `Error` the passed contexts will be ignored
func Wrap(err error, params map[string]string) Error {
	if err == nil {
		return nil
	}
	switch err := err.(type) {
	case *errorImpl:
		return err
	default:
		return errorFactory(ErrInternalService, err.Error(), params)
	}
}

// InternalService creates a new error to represent an internal service error.
// Only use internal service error if we know very little about the error. Most
// internal service errors will come from `Wrap`ing a vanilla `error` interface
func InternalService(code, message string, params map[string]string) Error {
	return errorFactory(fmt.Sprintf("%s.%s", ErrInternalService, code), message, params)
}

// BadRequest creates a new error to represent an error caused by the client sending
// an invalid request. This is non-retryable unless the request is modified.
func BadRequest(code, message string, params map[string]string) Error {
	return errorFactory(fmt.Sprintf("%s.%s", ErrBadRequest, code), message, params)
}

// BadResponse creates a new error representing a failure to response with a valid response
// Examples of this would be a handler returning an invalid message format
func BadResponse(code, message string, params map[string]string) Error {
	return errorFactory(fmt.Sprintf("%s.%s", ErrBadResponse, code), message, params)
}

// Timeout creates a new error representing a timeout from client to server
func Timeout(code, message string, params map[string]string) Error {
	return errorFactory(fmt.Sprintf("%s.%s", ErrTimeout, code), message, params)
}

// NotFound creates a new error representing a resource that cannot be found. In some
// cases this is not an error, and would be better represented by a zero length slice of elements
func NotFound(code, message string, params map[string]string) Error {
	return errorFactory(fmt.Sprintf("%s.%s", ErrNotFound, code), message, params)
}

// Forbidden creates a new error representing a resource that cannot be accessed with
// the current authorisation credentials. The user may need authorising, or if authorised,
// may not be permitted to perform this action
func Forbidden(code, message string, params map[string]string) Error {
	return errorFactory(fmt.Sprintf("%s.%s", ErrForbidden, code), message, params)
}

// Unauthorized creates a new error indicating that authentication is required,
// but has either failed or not been provided.
func Unauthorized(code, message string, params map[string]string) Error {
	return errorFactory(fmt.Sprintf("%s.%s", ErrUnauthorized, code), message, params)
}

// errorConstructor returns a `*Error` with the specified code, message and context. The main work
// consists of managing the map[string]string's at the end of the arguments list.
// In practice we only ever pass in two: the first one is public and will be sent to the client,
// the second one is private and can contain internal information that is useful for debugging
func errorFactory(code string, message string, params map[string]string) Error {
	err := &errorImpl{
		code:    code,
		message: message,
		params:  map[string]string{},
	}
	if params != nil {
		err.params = params
	}
	// TODO pass in context.Context

	// Build stack and skip first three lines:
	//  - stack.go BuildStack()
	//  - errors.go errorFactory()
	//  - errors.go public constructor method
	err.stackFrames = stack.BuildStack(3)

	// ... ignore all remaining map[string]string
	return err
}
