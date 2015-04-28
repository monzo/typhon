package errors

import (
	"testing"

	"github.com/b2aio/typhon/errors/stack"
	pe "github.com/b2aio/typhon/proto/error"
	"github.com/stretchr/testify/assert"
)

func TestMarshalNilError(t *testing.T) {
	var input *Error // nil
	protoError := Marshal(input)

	assert.NotNil(t, protoError)
	assert.Equal(t, ErrUnknown, int(protoError.Code))
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
	// test blank error
	{
		&Error{},
		&pe.Error{},
	},
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
			PublicContext: map[string]string{
				"something": "hullo",
			},
			PrivateContext: map[string]string{
				"something else": "bye bye",
			},
			Stack: []stack.Frame{
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
		&pe.Error{
			Code:    ErrTimeout,
			Message: "omg help plz",
			PublicContext: map[string]string{
				"something": "hullo",
			},
			PrivateContext: map[string]string{
				"something else": "bye bye",
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
		assert.Equal(t, tc.protoErr.PublicContext, protoError.PublicContext)
		assert.Equal(t, tc.protoErr.PrivateContext, protoError.PrivateContext)
	}
}

// these are separate from above because the marshaling and unmarshaling isn'y symmetric.
// protobuf turns empty maps[string]string into nil :(
var unmarshalTestCases = []struct {
	platErr  *Error
	protoErr *pe.Error
}{
	{
		&Error{
			PublicContext:  map[string]string{},
			PrivateContext: map[string]string{},
		},
		&pe.Error{},
	},
	{
		&Error{
			PublicContext:  map[string]string{},
			PrivateContext: map[string]string{},
		},
		&pe.Error{
			Code: ErrUnknown,
		},
	},
	{
		&Error{
			Code:    ErrTimeout,
			Message: "omg help plz",
			PublicContext: map[string]string{
				"something": "hullo",
			},
			PrivateContext: map[string]string{
				"something else": "bye bye",
			},
			Stack: []stack.Frame{
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
		&pe.Error{
			Code:    ErrTimeout,
			Message: "omg help plz",
			PublicContext: map[string]string{
				"something": "hullo",
			},
			PrivateContext: map[string]string{
				"something else": "bye bye",
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
			Code:           ErrForbidden,
			Message:        "NO. FORBIDDEN",
			PublicContext:  map[string]string{},
			PrivateContext: map[string]string{},
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
		assert.Equal(t, tc.platErr.PublicContext, platErr.PublicContext)
		assert.Equal(t, tc.platErr.PrivateContext, platErr.PrivateContext)
	}
}
