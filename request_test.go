package typhon

import (
	"bytes"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
