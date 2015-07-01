package errors

import (
	"fmt"

	"github.com/mondough/typhon/errors/stack"
)

type Error struct {
	Code           int
	Message        string
	PublicContext  map[string]string
	PrivateContext map[string]string
	Stack          stack.Stack
}

// Generic error codes. Each of these has their own constructor for convenience.
// You can use any integer as a code, just use the `New` method.
const (
	ErrUnknown         = 0
	ErrInternalService = 1
	ErrBadRequest      = 2
	ErrBadResponse     = 3
	ErrForbidden       = 4
	ErrUnauthorized    = 5
	ErrNotFound        = 6
	ErrTimeout         = 7
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
	for _, frame := range p.Stack {
		stackStr = fmt.Sprintf("%s\n  %s:%d in %s", stackStr, frame.Filename, frame.Line, frame.Method)
	}
	return stackStr
}

// New creates a new error for you. Use this if you want to pass along a custom error code.
// Otherwise use the handy shorthand factories below
func New(code int, message string, context ...map[string]string) *Error {
	return errorFactory(code, message, context...)
}

// Wrap takes any error interface and wraps it into an Error.
// This is useful because an Error contains lots of useful goodies, like the stacktrace of the error.
// NOTE: If `err` is already an `Error` the passed contexts will be ignored
func Wrap(err error, context ...map[string]string) *Error {
	if err == nil {
		return nil
	}
	switch err := err.(type) {
	case *Error:
		return err
	default:
		return errorFactory(ErrInternalService, err.Error(), context...)
	}
}

// InternalService creates a new error to represent an internal service error.
// Only use internal service error if we know very little about the error. Most
// internal service errors will come from `Wrap`ing a vanilla `error` interface
func InternalService(message string, context ...map[string]string) *Error {
	return errorFactory(ErrInternalService, message, context...)
}

// BadRequest creates a new error to represent an error caused by the client sending
// an invalid request. This is non-retryable unless the request is modified.
func BadRequest(message string, context ...map[string]string) *Error {
	return errorFactory(ErrBadRequest, message, context...)
}

// BadResponse creates a new error representing a failure to response with a valid response
// Examples of this would be a handler returning an invalid message format
func BadResponse(message string, context ...map[string]string) *Error {
	return errorFactory(ErrBadResponse, message, context...)
}

// Timeout creates a new error representing a timeout from client to server
func Timeout(message string, context ...map[string]string) *Error {
	return errorFactory(ErrTimeout, message, context...)
}

// NotFound creates a new error representing a resource that cannot be found. In some
// cases this is not an error, and would be better represented by a zero length slice of elements
func NotFound(message string, context ...map[string]string) *Error {
	return errorFactory(ErrNotFound, message, context...)
}

// Forbidden creates a new error representing a resource that cannot be accessed with
// the current authorisation credentials. The user may need authorising, or if authorised,
// may not be permitted to perform this action
func Forbidden(message string, context ...map[string]string) *Error {
	return errorFactory(ErrForbidden, message, context...)
}

// Unauthorized creates a new error indicating that authentication is required,
// but has either failed or not been provided.
func Unauthorized(message string, context ...map[string]string) *Error {
	return errorFactory(ErrUnauthorized, message, context...)
}

// errorConstructor returns a `*Error` with the specified code, message and context. The main work
// consists of managing the map[string]string's at the end of the arguments list.
// In practice we only ever pass in two: the first one is public and will be sent to the client,
// the second one is private and can contain internal information that is useful for debugging
func errorFactory(code int, message string, context ...map[string]string) *Error {
	err := &Error{
		Code:           code,
		Message:        message,
		PrivateContext: map[string]string{},
		PublicContext:  map[string]string{},
	}
	// The first context map is the PublicContext
	if len(context) > 0 && context[0] != nil {
		err.PublicContext = context[0]
	}
	// The second context map is the privateContext
	if len(context) > 1 && context[1] != nil {
		err.PrivateContext = context[1]
	}

	// Build stack and skip first three lines:
	//  - stack.go BuildStack()
	//  - errors.go errorFactory()
	//  - errors.go public constructor method
	err.Stack = stack.BuildStack(3)

	// ... ignore all remaining map[string]string
	return err
}
