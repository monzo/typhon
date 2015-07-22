package message

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type jsonStruct struct {
	Foo string `json:"foo"`
	Bar int    `json:"bar"`
}

func TestJSONMarshaling_Struct(t *testing.T) {
	impl := &jsonStruct{
		Foo: "Bar",
		Bar: 1}
	req := NewRequest()
	req.SetBody(impl)
	require.NoError(t, JSONMarshaler().MarshalBody(req), "Error marshaling")

	expectedPayload, err := json.Marshal(impl)
	require.NoError(t, err, "Error marshaling (direct to JSON)")
	assert.Equal(t, expectedPayload, req.Payload())

	req2 := req.Copy()
	req2.SetBody(nil)
	require.NoError(t, JSONUnmarshaler(&jsonStruct{}).UnmarshalPayload(req2), "Error unmarshaling")
	assert.Equal(t, impl, req2.Body())
}

func TestJSONMarshaling_Map(t *testing.T) {
	impl := map[string]interface{}{
		"foo": "bar",
		"bar": float64(1)}
	req := NewRequest()
	req.SetBody(impl)
	require.NoError(t, JSONMarshaler().MarshalBody(req), "Error marshaling")

	expectedPayload, err := json.Marshal(impl)
	require.NoError(t, err, "Error marshaling (direct to JSON)")
	assert.Equal(t, expectedPayload, req.Payload())

	req2 := req.Copy()
	req2.SetBody(nil)
	require.NoError(t, JSONUnmarshaler(map[string]interface{}(nil)).UnmarshalPayload(req2), "Error unmarshaling")
	assert.Equal(t, impl, req2.Body())
}
