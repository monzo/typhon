package errors

import (
	"testing"

	"github.com/mondough/typhon/errors/stack"
	pe "github.com/mondough/typhon/proto/error"
	"github.com/stretchr/testify/assert"
)

func TestMarshalNilError(t *testing.T) {
	var input *Error // nil
	protoError := Marshal(input)

	assert.NotNil(t, protoError)
	assert.Equal(t, ErrUnknown, protoError.Code)
	assert.NotEmpty(t, protoError.Message)
}

func TestUnmarshalNilError(t *testing.T) {
	var input *pe.Error // nil
	platError := Unmarshal(input)

	assert.NotNil(t, platError)
	assert.Equal(t, ErrUnknown, platError.Code)
	assert.Equal(t, "Nil error unmarshalled!", platError.Message)
}

// marshalTestCases represents a set of error formats
// which should be marshaled
var marshalTestCases = []struct {
	platErr  *Error
	protoErr *pe.Error
}{
	// confirm blank errors (shouldn't be possible) are UNKNOWN
	{
		&Error{},
		&pe.Error{
			Code: ErrUnknown,
		},
	},
	// normal cases
	{
		&Error{
			Code:    ErrTimeout,
			Message: "omg help plz",
			Params: map[string]string{
				"something": "hullo",
			},
			StackFrames: []*stack.Frame{
				&stack.Frame{Filename: "some file", Method: "someMethod", Line: 123},
				&stack.Frame{Filename: "another file", Method: "someOtherMethod", Line: 1},
			},
		},
		&pe.Error{
			Code:    ErrTimeout,
			Message: "omg help plz",
			Params: map[string]string{
				"something": "hullo",
			},
			Stack: []*pe.StackFrame{
				{
					Filename: "some file",
					Line:     123,
					Method:   "someMethod",
				},
				{
					Filename: "another file",
					Line:     1,
					Method:   "someOtherMethod",
				},
			},
		},
	},
	{
		&Error{
			Code:    ErrForbidden,
			Message: "NO. FORBIDDEN",
		},
		&pe.Error{
			Code:    ErrForbidden,
			Message: "NO. FORBIDDEN",
		},
	},
}

func TestMarshal(t *testing.T) {
	for _, tc := range marshalTestCases {
		protoError := Marshal(tc.platErr)
		assert.Equal(t, tc.protoErr.Code, protoError.Code)
		assert.Equal(t, tc.protoErr.Message, protoError.Message)
		assert.Equal(t, tc.protoErr.Params, protoError.Params)
	}
}

// these are separate from above because the marshaling and unmarshaling isn't symmetric.
// protobuf turns empty maps[string]string into nil :(
var unmarshalTestCases = []struct {
	platErr  *Error
	protoErr *pe.Error
}{
	{
		New("", "", nil),
		&pe.Error{},
	},
	{
		New("", "", nil),
		&pe.Error{
			Code: ErrUnknown,
		},
	},
	{
		&Error{
			Code:    ErrTimeout,
			Message: "omg help plz",
			Params: map[string]string{
				"something": "hullo",
			},
			StackFrames: []*stack.Frame{
				&stack.Frame{Filename: "some file", Method: "someMethod", Line: 123},
				&stack.Frame{Filename: "another file", Method: "someOtherMethod", Line: 1},
			},
		},
		&pe.Error{
			Code:    ErrTimeout,
			Message: "omg help plz",
			Params: map[string]string{
				"something": "hullo",
			},
			Stack: []*pe.StackFrame{
				{
					Filename: "some file",
					Line:     123,
					Method:   "someMethod",
				},
				{
					Filename: "another file",
					Line:     1,
					Method:   "someOtherMethod",
				},
			},
		},
	},
	{
		&Error{
			Code:    ErrForbidden,
			Message: "NO. FORBIDDEN",
			Params:  map[string]string{},
		},
		&pe.Error{
			Code:    ErrForbidden,
			Message: "NO. FORBIDDEN",
		},
	},
}

func TestUnmarshal(t *testing.T) {
	for _, tc := range unmarshalTestCases {
		platErr := Unmarshal(tc.protoErr)
		assert.Equal(t, tc.platErr.Code, platErr.Code)
		assert.Equal(t, tc.platErr.Message, platErr.Message)
		assert.Equal(t, tc.platErr.Params, platErr.Params)
	}
}
