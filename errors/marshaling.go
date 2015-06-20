package errors

import (
	"github.com/obeattie/typhon/errors/stack"
	pe "github.com/obeattie/typhon/proto/error"
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
	return &pe.Error{
		Code:           int32(e.Code),
		Message:        e.Message,
		PublicContext:  e.PublicContext,
		PrivateContext: e.PrivateContext,
		Stack:          stackToProto(e.Stack),
	}
}

// Unmarshal a protobuf error into a local error
func Unmarshal(p *pe.Error) *Error {
	if p == nil {
		return &Error{
			Message: "Nil error unmarshalled!",
		}
	}
	// empty map[string]string come out as nil. thanks proto.
	publicContext := p.PublicContext
	if publicContext == nil {
		publicContext = map[string]string{}
	}
	privateContext := p.PrivateContext
	if privateContext == nil {
		privateContext = map[string]string{}
	}
	return &Error{
		Code:           int(p.Code),
		Message:        p.Message,
		PublicContext:  publicContext,
		PrivateContext: privateContext,
		Stack:          protoToStack(p.Stack),
	}
}

// stackToProto converts a stack.Stack and returns a slice of *pe.StackFrame
func protoToStack(protoStack []*pe.StackFrame) stack.Stack {
	if protoStack == nil {
		return stack.Stack{}
	}

	s := make(stack.Stack, 0, len(protoStack))
	for _, frame := range protoStack {
		s = append(s, stack.Frame{
			Filename: frame.Filename,
			Line:     int(frame.Line),
			Method:   frame.Method,
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
