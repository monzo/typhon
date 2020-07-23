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
	"strconv"
	"strings"

	"github.com/monzo/terrors/stack"
)

// Error is terror's error. It implements Go's error interface.
type Error struct {
	Code        string            `json:"code"`
	Message     string            `json:"message"`
	Params      map[string]string `json:"params"`
	StackFrames stack.Stack       `json:"stack"`

	// Cause is the initial cause of this error, and will be populated
	// when using the Propagate function. This is intentionally not exported
	// so that we don't serialize causes and send them across process boundaries.
	// The cause refers to the cause of the error within a given process, and you
	// should not expect it to contain information about terrors from other downstream
	// processes.
	cause error
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

// Error returns a string message of the error.
// It will contain the code and error message. If there is a causal chain, the
// message from each error in the chain will be added to the output.
func (p *Error) Error() string {
	if p.cause == nil {
		// Not sure if the empty code/message cases actually happen, but to be safe, defer to
		// the 'old' error message if there is no cause present (i.e. we're not using
		// new wrapping functionality)
		return p.legacyErrString()
	}
	var next error = p
	output := strings.Builder{}
	output.WriteString(p.Code)
	for next != nil {
		output.WriteString(": ")
		switch typed := next.(type) {
		case *Error:
			output.WriteString(typed.Message)
			next = typed.cause
		case error:
			output.WriteString(typed.Error())
			next = nil
		}
	}
	return output.String()
}

func (p *Error) legacyErrString() string {
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

// Unwrap retruns the cause of the error. It may be nil.
// WARNING: This function is considered experimental, and may be changed without notice.
func (p *Error) Unwrap() error {
	return p.cause
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

// LogMetadata implements the logMetadataProvider interface in the slog library which means that
// the error params will automatically be merged with the slog metadata.
// Additionally we put stack data in here for slog use.
func (p *Error) LogMetadata() map[string]string {
	if len(p.StackFrames) == 0 {
		return p.Params
	}

	// Attempt to find a frame that isn't within the terrors library.
	var frames []*stack.Frame
	for _, f := range p.StackFrames {
		if !strings.HasPrefix(f.Method, "terrors.") {
			frames = append(frames, f)
		}
	}
	if len(frames) == 0 {
		return p.Params
	}

	stackPCs := make([]string, len(frames))
	for i, f := range frames {
		stackPCs[i] = strconv.FormatUint(uint64(f.PC), 10)
	}

	logParams := map[string]string{
		"terrors_file":     frames[0].Filename,
		"terrors_function": frames[0].Method,
		"terrors_line":     strconv.Itoa(frames[0].Line),
		"terrors_pc":       strconv.FormatUint(uint64(frames[0].PC), 10),
		"terrors_stack":    strings.Join(stackPCs, ","),
	}

	for key, value := range p.Params {
		logParams[key] = value
	}

	return logParams
}

// New creates a new error for you. Use this if you want to pass along a custom error code.
// Otherwise use the handy shorthand factories below
func New(code string, message string, params map[string]string) *Error {
	return errorFactory(code, message, params)
}

// NewInternalWithCause creates a new Terror from an existing error.
// The new error will always have the code `ErrInternalService`. The original
// error is attached as the `cause`, and can be tested with the `Is` function.
// WARNING: This function is considered experimental, and may be changed without notice.
func NewInternalWithCause(err error, message string, params map[string]string, subCode string) *Error {
	newErr := errorFactory(errCode(ErrInternalService, subCode), message, params)
	newErr.cause = err
	return newErr
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
// you can match the error on different levels e.g. dotted codes `bad_request` or `bad_request.missing_param`. Each
// dotted part can be passed as a separate argument e.g. `terr.PrefixMatches(terrors.ErrBadRequest, "missing_param")`
// is the same as `terr.PrefixMatches("bad_request.missing_param")`
func (p *Error) PrefixMatches(prefixParts ...string) bool {
	prefix := strings.Join(prefixParts, ".")

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
// `bad_request.missing_param`. Each dotted part can be passed as a separate argument
// e.g. `terrors.PrefixMatches(terr, terrors.ErrBadRequest, "missing_param")` is the same as
// terrors.PrefixMatches(terr, "bad_request.missing_param")`
func PrefixMatches(err error, prefixParts ...string) bool {
	if terr, ok := Wrap(err, nil).(*Error); ok {
		return terr.PrefixMatches(prefixParts...)
	}

	return false
}

// Propagate adds context to an existing error.
// If the error given is not already a terror, a new terror is created.
// A new stack trace is taken with each call to propagate. This is most likely only useful
// for the first call, so callers should ensure they examine the causal chain of errors to
// select the stack trace with the most details.
// WARNING: This function is considered experimental, and may be changed without notice.
func Propagate(err error, context string, params map[string]string) error {
	switch err := err.(type) {
	case *Error:
		terr := addParams(err, params)
		terr.Message = context
		terr.cause = err
		terr.StackFrames = stack.BuildStack(2) // 2 skips 'BuildStack' and 'Propagate'
		return terr
	default:
		return NewInternalWithCause(err, context, params, "")
	}
}

// Is checks whether an error is a given code. Similarly to `errors.Is`,
// this unwinds the error stack and checks each underlying error for the code.
// If any match, this returns true.
// We prefer this over using a method receiver on the terrors Error, as the function
// signature requires an error to test against, and checking against terrors would
// requite creating a new terror with the specific code.
// WARNING: This function is considered experimental, and may be changed without notice.
func Is(err error, code ...string) bool {
	switch err := err.(type) {
	case *Error:
		if err.PrefixMatches(code...) {
			return true
		}
		next := err.Unwrap()
		if next == nil {
			return false
		}
		return Is(next, code...)
	default:
		return false
	}
}
