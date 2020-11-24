package terrors

import (
	"strings"

	"github.com/monzo/terrors/stack"
)

var retryable bool = true
var notRetryable bool = false

// Wrap takes any error interface and wraps it into an Error.
// This is useful because an Error contains lots of useful goodies, like the stacktrace of the error.
// NOTE: If `err` is already an `Error`, it will add the params passed in to the params of the Error
func Wrap(err error, params map[string]string) error {
	return WrapWithCode(err, params, ErrInternalService)
}

// WrapWithCode wraps an error with a custom error code. If `err` is already
// an `Error`, it will add the params passed in to the params of the error
func WrapWithCode(err error, params map[string]string, code string) error {
	if err == nil {
		return nil
	}
	switch err := err.(type) {
	case *Error:
		return addParams(err, params)
	default:
		return errorFactory(code, err.Error(), params)
	}
}

// InternalService creates a new error to represent an internal service error.
// Only use internal service error if we know very little about the error. Most
// internal service errors will come from `Wrap`ing a vanilla `error` interface
func InternalService(code, message string, params map[string]string) *Error {
	return errorFactory(errCode(ErrInternalService, code), message, params)
}

// BadRequest creates a new error to represent an error caused by the client sending
// an invalid request. This is non-retryable unless the request is modified.
func BadRequest(code, message string, params map[string]string) *Error {
	return errorFactory(errCode(ErrBadRequest, code), message, params)
}

// BadResponse creates a new error representing a failure to response with a valid response
// Examples of this would be a handler returning an invalid message format
func BadResponse(code, message string, params map[string]string) *Error {
	return errorFactory(errCode(ErrBadResponse, code), message, params)
}

// Timeout creates a new error representing a timeout from client to server
func Timeout(code, message string, params map[string]string) *Error {
	return errorFactory(errCode(ErrTimeout, code), message, params)
}

// NotFound creates a new error representing a resource that cannot be found. In some
// cases this is not an error, and would be better represented by a zero length slice of elements
func NotFound(code, message string, params map[string]string) *Error {
	return errorFactory(errCode(ErrNotFound, code), message, params)
}

// Forbidden creates a new error representing a resource that cannot be accessed with
// the current authorisation credentials. The user may need authorising, or if authorised,
// may not be permitted to perform this action
func Forbidden(code, message string, params map[string]string) *Error {
	return errorFactory(errCode(ErrForbidden, code), message, params)
}

// Unauthorized creates a new error indicating that authentication is required,
// but has either failed or not been provided.
func Unauthorized(code, message string, params map[string]string) *Error {
	return errorFactory(errCode(ErrUnauthorized, code), message, params)
}

// PreconditionFailed creates a new error indicating that one or more conditions
// given in the request evaluated to false when tested on the server.
func PreconditionFailed(code, message string, params map[string]string) *Error {
	return errorFactory(errCode(ErrPreconditionFailed, code), message, params)
}

// RateLimited creates a new error indicating that the request has been rate-limited,
// and that the caller should back-off.
func RateLimited(code, message string, params map[string]string) *Error {
	return errorFactory(errCode(ErrRateLimited, code), message, params)
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

		err.IsRetryable = &notRetryable
		for _, c := range retryableCodes {
			if PrefixMatches(err, c) {
				err.IsRetryable = &retryable
			}
		}
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

func errCode(prefix, code string) string {
	if code == "" {
		return prefix
	}
	if prefix == "" {
		return code
	}
	return strings.Join([]string{prefix, code}, ".")
}
