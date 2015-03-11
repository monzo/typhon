package errors

import (
	"testing"

	pe "github.com/b2aio/typhon/proto/error"
	"github.com/stretchr/testify/assert"
)

// errorTypeTestCases matches error types between formats
var errorTypeTestCases = []struct {
	platErr  ErrorType
	protoErr pe.ErrorType
}{
	{ErrUnknown, pe.ErrorType_UNKNOWN},
	{ErrInternalService, pe.ErrorType_INTERNAL_SERVICE},
	{ErrBadRequest, pe.ErrorType_BAD_REQUEST},
	{ErrBadResponse, pe.ErrorType_BAD_RESPONSE},
	{ErrTimeout, pe.ErrorType_TIMEOUT},
	{ErrNotFound, pe.ErrorType_NOT_FOUND},
	{ErrForbidden, pe.ErrorType_FORBIDDEN},
	{ErrUnauthorized, pe.ErrorType_UNAUTHORIZED},
}

func TestMarshalErrorTypes(t *testing.T) {

	// Assert types are interchanged correctly
	for _, tc := range errorTypeTestCases {
		platErr := &ServiceError{
			errorType: tc.platErr,
		}
		protoError := Marshal(platErr)
		assert.Equal(t, tc.protoErr, protoError.Type)
	}

	// Sneakily assert we've checked every case defined in the proto
	assert.Equal(t, len(errorTypeTestCases), len(pe.ErrorType_name))
}

func TestUnmarshalErrorTypes(t *testing.T) {

	// Assert types are interchanged correctly
	for _, tc := range errorTypeTestCases {
		platErr := &ServiceError{
			errorType: tc.platErr,
		}
		protoError := Marshal(platErr)
		assert.Equal(t, tc.protoErr, protoError.Type)
	}

	// Sneakily assert we've checked every case defined in the proto
	assert.Equal(t, len(errorTypeTestCases), len(pe.ErrorType_name))
}

func TestMarshalNilError(t *testing.T) {
	var input *ServiceError // nil
	protoError := Marshal(input)

	assert.NotNil(t, protoError)
	assert.Equal(t, pe.ErrorType_UNKNOWN, protoError.Type)
	assert.NotEmpty(t, protoError.Code)
	assert.NotEmpty(t, protoError.Description)
}

func TestUnmarshalNilError(t *testing.T) {
	var input *pe.Error // nil
	platError := Unmarshal(input)

	assert.NotNil(t, platError)
	assert.Equal(t, ErrUnknown, platError.Type())
	assert.Empty(t, platError.Code())
	assert.Empty(t, platError.Description())
}

// interchangingErrorTestCases represents a set of error formats
// which should be converted between
var interchangableErrorTestCases = []struct {
	platErr  *ServiceError
	protoErr *pe.Error
}{
	// test blank error
	{
		&ServiceError{},
		&pe.Error{},
	},
	// confirm blank errors (shouldn't be possible) are UNKNOWN
	{
		&ServiceError{},
		&pe.Error{
			Type: pe.ErrorType_UNKNOWN,
		},
	},
	// normal cases
	{
		&ServiceError{
			errorType:   ErrInternalService,
			code:        "some.error",
			description: "omg help plz",
			clientCode:  123,
			publicContext: map[string]string{
				"something": "hullo",
			},
			privateContext: map[string]string{
				"something else": "bye bye",
			},
		},
		&pe.Error{
			Type:        pe.ErrorType_INTERNAL_SERVICE,
			Code:        "some.error",
			Description: "omg help plz",
			ClientCode:  123,
			PublicContext: map[string]string{
				"something": "hullo",
			},
			PrivateContext: map[string]string{
				"something else": "bye bye",
			},
		},
	},
	{
		&ServiceError{
			errorType:   ErrForbidden,
			code:        "denied.access",
			description: "NO. FORBIDDEN",
		},
		&pe.Error{
			Type:        pe.ErrorType_FORBIDDEN,
			Code:        "denied.access",
			Description: "NO. FORBIDDEN",
		},
	},
}

func TestMarshal(t *testing.T) {
	for _, tc := range interchangableErrorTestCases {
		protoError := Marshal(tc.platErr)
		assert.Equal(t, tc.protoErr.Type, protoError.Type)
		assert.Equal(t, tc.protoErr.Code, protoError.Code)
		assert.Equal(t, tc.protoErr.Description, protoError.Description)
	}
}

func TestUnmarshal(t *testing.T) {
	for _, tc := range interchangableErrorTestCases {
		platErr := Unmarshal(tc.protoErr)
		assert.Equal(t, tc.platErr.Type(), platErr.Type())
		assert.Equal(t, tc.platErr.Code(), platErr.Code())
		assert.Equal(t, tc.platErr.Description(), platErr.Description())
	}
}
