package terrors

import (
	pe "github.com/monzo/terrors/proto"
	"github.com/monzo/terrors/stack"
)

// Marshal an error into a protobuf for transmission
func Marshal(e *Error) *pe.Error {
	// Account for nil errors
	if e == nil {
		return &pe.Error{
			Code:    ErrUnknown,
			Message: "Unknown error, nil error marshalled",
		}
	}

	retryable := &pe.BoolValue{}
	if e.IsRetryable != nil {
		retryable.Value = *e.IsRetryable
	}

	err := &pe.Error{
		Code:      e.Code,
		Message:   e.Message,
		Stack:     stackToProto(e.StackFrames),
		Params:    e.Params,
		Retryable: retryable,
	}
	if err.Code == "" {
		err.Code = ErrUnknown
	}
	return err
}

// Unmarshal a protobuf error into a local error
func Unmarshal(p *pe.Error) *Error {
	if p == nil {
		return &Error{
			Code:    ErrUnknown,
			Message: "Nil error unmarshalled!",
			Params:  map[string]string{},
		}
	}

	var retryable *bool
	if p.Retryable != nil {
		retryable = &p.Retryable.Value
	}

	err := &Error{
		Code:        p.Code,
		Message:     p.Message,
		StackFrames: protoToStack(p.Stack),
		Params:      p.Params,
		IsRetryable: retryable,
	}
	if err.Code == "" {
		err.Code = ErrUnknown
	}
	// empty map[string]string come out as nil. thanks proto.
	if err.Params == nil {
		err.Params = map[string]string{}
	}
	return err
}

// protoToStack converts a slice of *pe.StackFrame and returns a stack.Stack
func protoToStack(protoStack []*pe.StackFrame) stack.Stack {
	if protoStack == nil {
		return stack.Stack{}
	}

	s := make(stack.Stack, 0, len(protoStack))
	for _, frame := range protoStack {
		s = append(s, &stack.Frame{
			Filename: frame.Filename,
			Method:   frame.Method,
			Line:     int(frame.Line),
		})
	}
	return s
}

// stackToProto converts a stack.Stack and returns a slice of *pe.StackFrame
func stackToProto(s stack.Stack) []*pe.StackFrame {
	if s == nil {
		return []*pe.StackFrame{}
	}

	protoStack := make([]*pe.StackFrame, 0, len(s))
	for _, frame := range s {
		protoStack = append(protoStack, &pe.StackFrame{
			Filename: frame.Filename,
			Line:     int32(frame.Line),
			Method:   frame.Method,
		})
	}
	return protoStack
}
