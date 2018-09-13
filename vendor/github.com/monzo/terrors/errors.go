// Package terrors implements an error wrapping library.
//
// Terrors are used to provide context to an error, offering a stack trace and
// user defined error parameters.
//
// Terrors can be used to wrap any object that satisfies the error interface:
//	terr := terrors.Wrap(err, map[string]string{"context": "my_context"})
//
// Terrors can be instantiated directly:
// 	err := terrors.New("not_found", "object not found", map[string]string{
//		"context": "my_context"
//	})
//
// Terrors offers built-in functions for instantiating Errors with common codes:
//	err := terrors.NotFound("config_file", "config file not found", map[string]string{
//		"context": my_context
//	})
package terrors

import (
	"fmt"
	"strings"

	"github.com/monzo/terrors/stack"
)

// Error is terror's error. It implements Go's error interface.
type Error struct {
	Code        string            `json:"code"`
	Message     string            `json:"message"`
	Params      map[string]string `json:"params"`
	StackFrames stack.Stack       `json:"stack"`
}

// Generic error codes. Each of these has their own constructor for convenience.
// You can use any string as a code, just use the `New` method.
const (
	ErrBadRequest         = "bad_request"
	ErrBadResponse        = "bad_response"
	ErrForbidden          = "forbidden"
	ErrInternalService    = "internal_service"
	ErrNotFound           = "not_found"
	ErrPreconditionFailed = "precondition_failed"
	ErrTimeout            = "timeout"
	ErrUnauthorized       = "unauthorized"
	ErrUnknown            = "unknown"
)

// Error returns a string message of the error. It is a concatenation of Code and Message params
// This means the Error implements the error interface
func (p *Error) Error() string {
	if p == nil {
		return ""
	}
	if p.Message == "" {
		return p.Code
	}
	if p.Code == "" {
		return p.Message
	}
	return fmt.Sprintf("%s: %s", p.Code, p.Message)
}

// StackString formats the stack as a beautiful string with newlines
func (p *Error) StackString() string {
	stackStr := ""
	for _, frame := range p.StackFrames {
		stackStr = fmt.Sprintf("%s\n  %s:%d in %s", stackStr, frame.Filename, frame.Line, frame.Method)
	}
	return stackStr
}

// VerboseString returns the error message, stack trace and params
func (p *Error) VerboseString() string {
	return fmt.Sprintf("%s\nParams: %+v\n%s", p.Error(), p.Params, p.StackString())
}

func (p *Error) Format(f fmt.State, c rune) {
	f.Write([]byte(p.Message))
}

// LogMetadata implements the logMetadataProvider interface in the slog library which means that
// the error params will automatically be merged with the slog metadata.
func (p *Error) LogMetadata() map[string]string {
	return p.Params
}

// New creates a new error for you. Use this if you want to pass along a custom error code.
// Otherwise use the handy shorthand factories below
func New(code string, message string, params map[string]string) *Error {
	return errorFactory(code, message, params)
}

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

func errCode(prefix, code string) string {
	if code == "" {
		return prefix
	}
	if prefix == "" {
		return code
	}
	return strings.Join([]string{prefix, code}, ".")
}

// addParams returns a new error with new params merged into the original error's
func addParams(err *Error, params map[string]string) *Error {
	copiedParams := make(map[string]string, len(err.Params)+len(params))
	for k, v := range err.Params {
		copiedParams[k] = v
	}
	for k, v := range params {
		copiedParams[k] = v
	}

	return &Error{
		Code:        err.Code,
		Message:     err.Message,
		Params:      copiedParams,
		StackFrames: err.StackFrames,
	}
}

// Matches returns whether the string returned from error.Error() contains the given param string. This means you can
// match the error on different levels e.g. dotted codes `bad_request` or `bad_request.missing_param` or even on the
// more descriptive message
func (p *Error) Matches(match string) bool {
	return strings.Contains(p.Error(), match)
}

// PrefixMatches returns whether the string returned from error.Error() starts with the given param string. This means
// you can match the error on different levels e.g. dotted codes `bad_request` or `bad_request.missing_param`.
func (p *Error) PrefixMatches(prefix string) bool {
	return strings.HasPrefix(p.Code, prefix)
}

// Matches returns true if the error is a terror error and the string returned from error.Error() contains the given
// param string. This means you can match the error on different levels e.g. dotted codes `bad_request` or
// `bad_request.missing_param` or even on the more descriptive message
func Matches(err error, match string) bool {
	if terr, ok := Wrap(err, nil).(*Error); ok {
		return terr.Matches(match)
	}

	return false
}

// PrefixMatches returns true if the error is a terror and the string returned from error.Error() starts with the
// given param string. This means you can match the error on different levels e.g. dotted codes `bad_request` or
// `bad_request.missing_param`.
func PrefixMatches(err error, prefix string) bool {
	if terr, ok := Wrap(err, nil).(*Error); ok {
		return terr.PrefixMatches(prefix)
	}

	return false
}
