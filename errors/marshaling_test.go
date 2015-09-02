package errors

import (
	"testing"

	"github.com/mondough/typhon/errors/stack"
	pe "github.com/mondough/typhon/proto/error"
	"github.com/stretchr/testify/assert"
)

func TestMarshalNilError(t *testing.T) {
	var input Error // nil
	protoError := Marshal(input)

	assert.NotNil(t, protoError)
	assert.Equal(t, ErrUnknown, protoError.Code)
	assert.NotEmpty(t, protoError.Message)
}

func TestUnmarshalNilError(t *testing.T) {
	var input *pe.Error // nil
	platError := Unmarshal(input)

	assert.NotNil(t, platError)
	assert.Equal(t, ErrUnknown, platError.Code())
	assert.Equal(t, "Nil error unmarshalled!", platError.Message())
}

// marshalTestCases represents a set of error formats
// which should be marshaled
var marshalTestCases = []struct {
	platErr  Error
	protoErr *pe.Error
}{
	// confirm blank errors (shouldn't be possible) are UNKNOWN
	{
		&errorImpl{},
		&pe.Error{
			Code: ErrUnknown,
		},
	},
	// normal cases
	{
		&errorImpl{
			code:    ErrTimeout,
			message: "omg help plz",
			params: map[string]string{
				"something": "hullo",
			},
			stackFrames: []stack.Frame{
				stack.NewFrame("some file", "someMethod", 123),
				stack.NewFrame("another file", "someOtherMethod", 1),
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
		&errorImpl{
			code:    ErrForbidden,
			message: "NO. FORBIDDEN",
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
	platErr  Error
	protoErr *pe.Error
}{
	{
		&errorImpl{
			params: map[string]string{},
		},
		&pe.Error{},
	},
	{
		&errorImpl{
			params: map[string]string{},
		},
		&pe.Error{
			Code: ErrUnknown,
		},
	},
	{
		&errorImpl{
			code:    ErrTimeout,
			message: "omg help plz",
			params: map[string]string{
				"something": "hullo",
			},
			stackFrames: []stack.Frame{
				stack.NewFrame("some file", "someMethod", 123),
				stack.NewFrame("another file", "someOtherMethod", 1),
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
		&errorImpl{
			code:    ErrForbidden,
			message: "NO. FORBIDDEN",
			params:  map[string]string{},
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
		assert.Equal(t, tc.platErr.Code(), platErr.Code())
		assert.Equal(t, tc.platErr.Message(), platErr.Message())
		assert.Equal(t, tc.platErr.Params(), platErr.Params())
	}
}
