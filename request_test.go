package typhon

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"google.golang.org/protobuf/proto"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/monzo/typhon/prototest"
)

// TestRequestDecodeCloses verifies that a request body is closed after calling Decode()
func TestRequestDecodeCloses(t *testing.T) {
	t.Parallel()
	req := NewRequest(nil, "GET", "/", nil)
	b := []byte("{\"a\":\"b\"}\n")
	r := newDoneReader(ioutil.NopCloser(bytes.NewReader(b)), -1)
	req.Body = r

	bout := map[string]string{}
	req.Decode(&bout)
	select {
	case <-r.closed:
	default:
		assert.Fail(t, "response body was not closed after Decode()")
	}
}

// TestRequestEncodeReader verifies that passing an io.Reader to request.Encode() uses it properly as the body, and
// does not attempt to encode it as JSON
func TestRequestEncodeReader(t *testing.T) {
	t.Parallel()

	// io.ReadCloser: this should be used with no modification
	rc := ioutil.NopCloser(strings.NewReader("hello world"))
	req := NewRequest(nil, "GET", "/", nil)
	req.Encode(rc)
	assert.Equal(t, req.Body, rc)
	assert.EqualValues(t, -1, req.ContentLength)
	assert.Empty(t, req.Header.Get("Content-Type"))

	// io.Reader: this should be wrapped in an ioutil.NopCloser
	r := strings.NewReader("hello world, again")
	req = NewRequest(nil, "GET", "/", nil)
	req.Encode(r)
	assert.EqualValues(t, -1, req.ContentLength)
	assert.Empty(t, req.Header.Get("Content-Type"))
	body, err := ioutil.ReadAll(req.Body)
	require.NoError(t, err)
	assert.Equal(t, []byte("hello world, again"), body)

	// an io.ReadCloser that happens to implement json.Marshaler should not be used directly and should be marshaled
	jm := jsonMarshalerReader{
		ReadCloser: ioutil.NopCloser(strings.NewReader("this should never see the light of day"))}
	req = NewRequest(nil, "GET", "/", nil)
	req.Encode(jm)
	assert.EqualValues(t, 3, req.ContentLength)
	assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
	body, err = ioutil.ReadAll(req.Body)
	require.NoError(t, err)
	assert.Equal(t, []byte("{}\n"), body)
}

func TestRequestEncodeProtobuf(t *testing.T) {
	g := &prototest.Greeting{
		Message:  "Hello world!",
		Priority: 1}

	protoContentForComparison, err := proto.Marshal(g)
	require.NoError(t, err)

	req := NewRequest(nil, "GET", "/", nil)
	req.EncodeAsProtobuf(g)

	bodyBytes, err := ioutil.ReadAll(req.Body)
	require.NoError(t, err)

	assert.Equal(t, "application/protobuf", req.Header.Get("Content-Type"))
	assert.EqualValues(t, bodyBytes, protoContentForComparison)

}

func TestRequestEncodeJSON(t *testing.T) {
	message := map[string]interface{} {
		"foo": "bar",
		"bar": 3,
	}

	jsonContentForComparison, err := jsonStreamMarshal(message)
	require.NoError(t, err)

	req := NewRequest(nil, "GET", "/", nil)
	req.Encode(message)

	bodyBytes, err := ioutil.ReadAll(req.Body)
	require.NoError(t, err)

	assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
	assert.EqualValues(t, bodyBytes, jsonContentForComparison)
}

func TestRequestSetMetadata(t *testing.T) {
	t.Parallel()

	ctx := AppendMetadataToContext(context.Background(), NewMetadata(map[string]string{
		"meta": "data",
	}))

	req := NewRequest(ctx, "GET", "/", nil)

	assert.Equal(t, []string{"data"}, req.Request.Header["meta"])
}


func jsonStreamMarshal(v interface{}) ([]byte, error) {
	var buffer bytes.Buffer
	writer := bufio.NewWriter(&buffer)

	err := json.NewEncoder(writer).Encode(v)

	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}
