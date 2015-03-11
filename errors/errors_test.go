package errors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type newError func(code, description string, context ...map[string]string) Error

func TestErrorConstructors(t *testing.T) {

	// mock out the ClientCodes registry
	ClientCodes = map[string]int{
		"boop.some.error":   101,
		"client.sent.cruft": 102,
	}

	testCases := []struct {
		constructor newError
		code        string
		description string
		contexts    []map[string]string

		expectedErrType        ErrorType
		expectedClientCode     int
		expectedPublicContext  map[string]string
		expectedPrivateContext map[string]string
	}{
		{
			InternalService, "boop.some.error", "oh crap", nil,
			ErrInternalService, 101, nil, nil,
		},
		{
			BadRequest, "client.sent.cruft", "please go away and rethink your life", nil,
			ErrBadRequest, 102, nil, nil,
		},
		{
			BadResponse, "server.responded.cruft", "server returned something crufty", nil,
			ErrBadResponse, DEFAULT_CLIENT_CODE, nil, nil,
		},
		{
			Timeout, "client.timed.out", "client timed out after the heat death of the universe", nil,
			ErrTimeout, DEFAULT_CLIENT_CODE, nil, nil,
		},
		{
			NotFound, "thing.notfound", "missing resource, resource doesn't exist", nil,
			ErrNotFound, DEFAULT_CLIENT_CODE, nil, nil,
		},
		{
			Forbidden, "access.denied", "user doesn't have permission to perform this action", nil,
			ErrForbidden, DEFAULT_CLIENT_CODE, nil, nil,
		},
		{
			Unauthorized, "authentication.required", "user needs to authenticate to perform this action", nil,
			ErrUnauthorized, DEFAULT_CLIENT_CODE, nil, nil,
		},
		{
			Unauthorized, "blub", "test public context", []map[string]string{{
				"some key":    "some value",
				"another key": "another value",
			}},
			ErrUnauthorized, DEFAULT_CLIENT_CODE, map[string]string{
				"some key":    "some value",
				"another key": "another value",
			}, nil,
		},
		{
			Unauthorized, "blub", "test public + private context", []map[string]string{{
				"some key": "some value",
			}, {
				"some private key": "woah cool",
			}},
			ErrUnauthorized, DEFAULT_CLIENT_CODE, map[string]string{
				"public key": "public value",
			}, map[string]string{
				"some private key": "woah cool",
			},
		},
	}

	for _, tc := range testCases {
		err := tc.constructor(tc.code, tc.description)
		assert.Equal(t, tc.code, err.Code())
		assert.Equal(t, tc.description, err.Error())
		assert.Equal(t, tc.expectedErrType, err.Type())
		assert.Equal(t, tc.expectedClientCode, err.ClientCode())
	}
}

func TestUnknownError(t *testing.T) {
	err := &ServiceError{}
	assert.Equal(t, ErrUnknown, err.Type())
}
