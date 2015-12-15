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

type jsonSlice []string

func TestJSONMarshaling_Struct(t *testing.T) {
	impl := jsonStruct{
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
	require.NoError(t, JSONUnmarshaler(jsonStruct{}).UnmarshalPayload(req2), "Error unmarshaling")
	assert.Equal(t, impl, req2.Body())
}

func TestJSONMarshaling_StructPointer(t *testing.T) {
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

	// Nil protocol
	req2 := req.Copy()
	req2.SetBody(nil)
	require.NoError(t, JSONUnmarshaler(map[string]interface{}(nil)).UnmarshalPayload(req2), "Error unmarshaling")
	assert.Equal(t, impl, req2.Body())

	// Non-nil protocol
	req2 = req.Copy()
	req2.SetBody(nil)
	require.NoError(t, JSONUnmarshaler(map[string]interface{}{}).UnmarshalPayload(req2), "Error unmarshaling")
	assert.Equal(t, impl, req2.Body())
}

func TestJSONMarshaling_MapString(t *testing.T) {
	impl := map[string]string{
		"foo": "bar",
		"bar": "foo"}
	req := NewRequest()
	req.SetBody(impl)
	require.NoError(t, JSONMarshaler().MarshalBody(req), "Error marshaling")

	expectedPayload, err := json.Marshal(impl)
	require.NoError(t, err, "Error marshaling (direct to JSON)")
	assert.Equal(t, expectedPayload, req.Payload())

	// Nil protocol
	req2 := req.Copy()
	req2.SetBody(nil)
	require.NoError(t, JSONUnmarshaler(map[string]string(nil)).UnmarshalPayload(req2), "Error unmarshaling")
	assert.Equal(t, impl, req2.Body())

	// Non-nil protocol
	req2 = req.Copy()
	req2.SetBody(nil)
	require.NoError(t, JSONUnmarshaler(map[string]string{}).UnmarshalPayload(req2), "Error unmarshaling")
	assert.Equal(t, impl, req2.Body())
}

func TestJSONMarshaling_Slice(t *testing.T) {
	impl := []string{"a", "ab", "abc", "abcd"}
	req := NewRequest()
	req.SetBody(impl)
	require.NoError(t, JSONMarshaler().MarshalBody(req), "Error marshaling")

	expectedPayload, err := json.Marshal(impl)
	require.NoError(t, err, "Error marshaling (direct to JSON)")
	assert.Equal(t, expectedPayload, req.Payload())

	// Nil protocol
	req2 := req.Copy()
	req2.SetBody(nil)
	require.NoError(t, JSONUnmarshaler([]string(nil)).UnmarshalPayload(req2), "Error unmarshaling")
	assert.Equal(t, impl, req2.Body())

	// Non-nil protocol
	req2.SetBody(nil)
	require.NoError(t, JSONUnmarshaler([]string{}).UnmarshalPayload(req2), "Error unmarshaling")
	assert.Equal(t, impl, req2.Body())
}

func TestJSONMarshaling_CustomSlice(t *testing.T) {
	impl := jsonSlice{"a", "ab", "abc", "abcd"}
	req := NewRequest()
	req.SetBody(impl)
	require.NoError(t, JSONMarshaler().MarshalBody(req), "Error marshaling")

	expectedPayload, err := json.Marshal(impl)
	require.NoError(t, err, "Error marshaling (direct to JSON)")
	assert.Equal(t, expectedPayload, req.Payload())

	// Nil protocol
	req2 := req.Copy()
	req2.SetBody(nil)
	require.NoError(t, JSONUnmarshaler(jsonSlice(nil)).UnmarshalPayload(req2), "Error unmarshaling")
	assert.Equal(t, impl, req2.Body())

	// Non-nil protocol
	req2 = req.Copy()
	req2.SetBody(nil)
	require.NoError(t, JSONUnmarshaler(jsonSlice{}).UnmarshalPayload(req2), "Error unmarshaling")
	assert.Equal(t, impl, req2.Body())
}

func TestJSONMarshaling_Interface(t *testing.T) {
	cases := []interface{}{
		[]interface{}{"a", "b", "c"},
		map[string]interface{}{"a": "b"},
		"a",
		float64(123.0)}

	for _, impl := range cases {
		req := NewRequest()
		req.SetBody(impl)
		require.NoError(t, JSONMarshaler().MarshalBody(req), "Error marshaling")

		expectedPayload, err := json.Marshal(impl)
		require.NoError(t, err, "Error marshaling (direct to JSON)")
		assert.Equal(t, expectedPayload, req.Payload())

		req2 := req.Copy()
		req2.SetBody(nil)
		require.NoError(t, JSONUnmarshaler(interface{}(nil)).UnmarshalPayload(req2), "Error unmarshaling")
		assert.Equal(t, impl, req2.Body())
	}
}
