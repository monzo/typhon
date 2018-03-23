package typhon

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
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
