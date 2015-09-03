package errors

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type newError func(code, message string, params map[string]string) *Error

func TestErrorConstructors(t *testing.T) {

	testCases := []struct {
		constructor  newError
		code         string
		message      string
		params       map[string]string
		expectedCode string
	}{
		{
			BadRequest, "service.foo", "please go away and rethink your life", nil, ErrBadRequest,
		},
		{
			BadResponse, "service.foo", "server returned something crufty", nil, ErrBadResponse,
		},
		{
			Timeout, "service.foo", "client timed out after the heat death of the universe", nil, ErrTimeout,
		},
		{
			NotFound, "service.foo", "missing resource, resource doesn't exist", nil, ErrNotFound,
		},
		{
			Forbidden, "service.foo", "user doesn't have permission to perform this action", nil, ErrForbidden,
		},
		{
			Unauthorized, "service.foo", "user needs to authenticate to perform this action", nil, ErrUnauthorized,
		},
		{
			Unauthorized, "service.foo", "test params", map[string]string{
				"some key":    "some value",
				"another key": "another value",
			}, ErrUnauthorized,
		},
	}

	for _, tc := range testCases {
		err := tc.constructor(tc.code, tc.message, tc.params)
		assert.Equal(t, fmt.Sprintf("%s.%s", tc.expectedCode, tc.code), err.Code)
		assert.Equal(t, tc.message, err.Error())
		if len(tc.params) > 0 {
			assert.Equal(t, tc.params, err.Params)
		}

	}
}

func TestNew(t *testing.T) {
	err := New("service.foo", "Some message", map[string]string{
		"public": "value",
	})

	assert.Equal(t, "service.foo", err.Code)
	assert.Equal(t, "Some message", err.Message)
	assert.Equal(t, map[string]string{
		"public": "value",
	}, err.Params)
}

func TestWrapWithWrappedErr(t *testing.T) {
	err := &Error{
		Code:    ErrForbidden,
		Message: "Some message",
		Params: map[string]string{
			"something old": "caesar",
		},
	}

	wrappedErr := Wrap(err, map[string]string{
		"something new": "a computer",
	}).(*Error)

	assert.Equal(t, wrappedErr, err)
	assert.Equal(t, ErrForbidden, wrappedErr.Code)
	assert.Equal(t, wrappedErr.Params, map[string]string{
		"something old": "caesar",
	})

}

func TestWrap(t *testing.T) {
	err := fmt.Errorf("Look here, an error")
	wrappedErr := Wrap(err, map[string]string{
		"blub": "dub",
	}).(*Error)

	assert.Equal(t, "Look here, an error", wrappedErr.Error())
	assert.Equal(t, "Look here, an error", wrappedErr.Message)
	assert.Equal(t, ErrInternalService, wrappedErr.Code)
	assert.Equal(t, wrappedErr.Params, map[string]string{
		"blub": "dub",
	})

}

func getNilErr() error {
	return Wrap(nil, nil)
}

func TestNilError(t *testing.T) {
	assert.Equal(t, getNilErr(), nil)
	assert.Nil(t, getNilErr())
	assert.Nil(t, Wrap(nil, nil))
}
