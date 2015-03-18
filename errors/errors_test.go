package errors

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type newError func(message string, context ...map[string]string) *Error

func TestErrorConstructors(t *testing.T) {

	testCases := []struct {
		constructor newError
		message     string
		contexts    []map[string]string

		expectedCode int
	}{
		{
			BadRequest, "please go away and rethink your life", nil, ErrBadRequest,
		},
		{
			BadResponse, "server returned something crufty", nil, ErrBadResponse,
		},
		{
			Timeout, "client timed out after the heat death of the universe", nil, ErrTimeout,
		},
		{
			NotFound, "missing resource, resource doesn't exist", nil, ErrNotFound,
		},
		{
			Forbidden, "user doesn't have permission to perform this action", nil, ErrForbidden,
		},
		{
			Unauthorized, "user needs to authenticate to perform this action", nil, ErrUnauthorized,
		},
		{
			Unauthorized, "test public context", []map[string]string{{
				"some key":    "some value",
				"another key": "another value",
			}}, ErrUnauthorized,
		},
		{
			Unauthorized, "test public + private context", []map[string]string{{
				"some key": "some value",
			}, {
				"some private key": "woah cool",
			}}, ErrUnauthorized,
		},
	}

	for _, tc := range testCases {
		err := tc.constructor(tc.message, tc.contexts...)
		assert.Equal(t, tc.expectedCode, err.Code)
		assert.Equal(t, tc.message, err.Error())
		if len(tc.contexts) >= 1 {
			assert.Equal(t, tc.contexts[0], err.PublicContext)
		}
		if len(tc.contexts) >= 2 {
			assert.Equal(t, tc.contexts[1], err.PrivateContext)
		}
	}
}

func TestNew(t *testing.T) {
	err := New(1234, "Some message", map[string]string{
		"public": "value",
	}, map[string]string{
		"private": "value",
	})

	assert.Equal(t, 1234, err.Code)
	assert.Equal(t, "Some message", err.Message)
	assert.Equal(t, map[string]string{
		"public": "value",
	}, err.PublicContext)
	assert.Equal(t, map[string]string{
		"private": "value",
	}, err.PrivateContext)
}

func TestWrapWithWrappedErr(t *testing.T) {
	err := &Error{
		Code:    ErrForbidden,
		Message: "Some message",
		PublicContext: map[string]string{
			"something old": "caesar",
		},
		PrivateContext: map[string]string{
			"something old and secret": "also caesar",
		},
	}

	wrappedErr := Wrap(err, map[string]string{
		"something new": "a computer",
	}, map[string]string{
		"something new and secret": "also a computer",
	})

	assert.Equal(t, wrappedErr, err)
	assert.Equal(t, ErrForbidden, wrappedErr.Code)
	assert.Equal(t, wrappedErr.PublicContext, map[string]string{
		"something old": "caesar",
		"something new": "a computer",
	})
	assert.Equal(t, wrappedErr.PrivateContext, map[string]string{
		"something old and secret": "also caesar",
		"something new and secret": "also a computer",
	})

}

func TestWrap(t *testing.T) {
	err := fmt.Errorf("Look here, an error")
	wrappedErr := Wrap(err, map[string]string{
		"blub": "dub",
	}, map[string]string{
		"dib": "dab",
	})

	assert.Equal(t, "Look here, an error", wrappedErr.Error())
	assert.Equal(t, "Look here, an error", wrappedErr.Message)
	assert.Equal(t, ErrInternalService, wrappedErr.Code)
	assert.Equal(t, wrappedErr.PublicContext, map[string]string{
		"blub": "dub",
	})
	assert.Equal(t, wrappedErr.PrivateContext, map[string]string{
		"dib": "dab",
	})

}
