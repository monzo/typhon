package message

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestCopying(t *testing.T) {
	r := NewRequest()
	r.SetId("id_123")
	r.SetService("service.foo")
	r.SetEndpoint("bar")
	r.SetHeader("X-Foo", "Bar")
	r.SetPayload([]byte("Mr. and Mrs. Payload lived happily ever after"))

	r2 := r.Copy()
	assert.Equal(t, "id_123", r2.Id())
	assert.Equal(t, "service.foo", r2.Service())
	assert.Equal(t, "bar", r2.Endpoint())
	assert.Equal(t, "Bar", r2.Headers()["X-Foo"])
	assert.Equal(t, "Mr. and Mrs. Payload lived happily ever after", string(r2.Payload()))

	// Mutate r2 and r1, see that changes don't affect each other
	r.SetId("id_1234")
	assert.Equal(t, "id_1234", r.Id())
	assert.Equal(t, "id_123", r2.Id())
	r2.SetId("id_12345")
	assert.Equal(t, "id_1234", r.Id())
	assert.Equal(t, "id_12345", r2.Id())
}
