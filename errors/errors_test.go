package errors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type newError func(code, description string, context ...string) Error

func TestErrorConstructors(t *testing.T) {
	testCases := []struct {
		constructor     newError
		expectedErrType ErrorType
		code            string
		description     string
	}{
		{
			NewInternalServerError,
			InternalServerError,
			"boop.some.error",
			"oh crap",
		},
		{
			NewBadRequestError,
			BadRequestError,
			"client.sent.some.cruft",
			"hey client, please go away and rethink your life",
		},
		{
			NewBadResponseError,
			BadResponseError,
			"server.responded.with.cruft",
			"server returned something that couldn't be marshaled or unmarshaled",
		},
		{
			NewTimeoutError,
			TimeoutError,
			"client.timed.out.waiting",
			"client timed out after the heat death of the universe",
		},
		{
			NewNotFoundError,
			NotFoundError,
			"thing.notfound",
			"missing resource, resource doesn't exist",
		},
	}

	for _, tc := range testCases {
		err := tc.constructor(tc.code, tc.description)
		assert.Equal(t, tc.code, err.Code())
		assert.Equal(t, tc.description, err.Error())
		assert.Equal(t, tc.expectedErrType, err.Type())
	}
}
