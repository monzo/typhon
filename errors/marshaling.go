package errors

import (
	"github.com/mondough/typhon/errors/stack"
	pe "github.com/mondough/typhon/proto/error"
)

// Marshal an error into a protobuf for transmission
func Marshal(e Error) *pe.Error {
	// Account for nil errors
	if e == nil {
		return &pe.Error{
			Code:    ErrUnknown,
			Message: "Unknown error, nil error marshalled",
		}
	}
	return &pe.Error{
		Code:    e.Code(),
		Message: e.Message(),
		Stack:   stackToProto(e.StackFrames()),
		Params:  e.Params(),
	}
}

// Unmarshal a protobuf error into a local error
func Unmarshal(p *pe.Error) Error {
	if p == nil {
		return &errorImpl{
			code:    ErrUnknown,
			message: "Nil error unmarshalled!",
			params:  map[string]string{},
		}
	}
	// empty map[string]string come out as nil. thanks proto.
	params := p.GetParams()
	if params == nil {
		params = map[string]string{}
	}
	return &errorImpl{
		code:        p.Code,
		message:     p.Message,
		stackFrames: protoToStack(p.Stack),
		params:      params,
	}
}

// protoToStack converts a slice of *pe.StackFrame and returns a stack.Stack
func protoToStack(protoStack []*pe.StackFrame) stack.Stack {
	if protoStack == nil {
		return stack.Stack{}
	}

	s := make(stack.Stack, 0, len(protoStack))
	for _, frame := range protoStack {
		s = append(s, stack.NewFrame(frame.Filename, frame.Method, int(frame.Line)))
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
			Filename: frame.Filename(),
			Line:     int32(frame.Line()),
			Method:   frame.Method(),
		})
	}
	return protoStack
}
