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
			BadRequest, "service.foo", "bad_request.service.foo", nil, ErrBadRequest,
		},
		{
			BadResponse, "service.foo", "bad_response.service.foo", nil, ErrBadResponse,
		},
		{
			Timeout, "service.foo", "timeout.service.foo", nil, ErrTimeout,
		},
		{
			NotFound, "service.foo", "not_found.service.foo", nil, ErrNotFound,
		},
		{
			Forbidden, "service.foo", "forbidden.service.foo", nil, ErrForbidden,
		},
		{
			Unauthorized, "service.foo", "unauthorized.service.foo", nil, ErrUnauthorized,
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
		assert.Equal(t, fmt.Sprintf("%s: %s", err.Code, tc.message), err.Error())
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

	assert.Equal(t, "internal_service: Look here, an error", wrappedErr.Error())
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

func TestMatches(t *testing.T) {
	err := &Error{
		Code:    "bad_request.missing_param.foo",
		Message: "You need to pass a value for foo; try passing foo=bar",
	}
	assert.True(t, err.Matches(ErrBadRequest))
	assert.True(t, err.Matches(ErrBadRequest+".missing_param"))
	assert.False(t, err.Matches(ErrInternalService))
	assert.False(t, err.Matches(ErrBadRequest+".missing_param.foo1"))
}

func TestMatches_Prefix(t *testing.T) {
	err := BadRequest("param_unknown", "Boop", nil)
	assert.False(t, err.Matches(ErrUnknown))
}
