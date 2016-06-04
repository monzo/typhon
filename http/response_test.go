package httpsvc

import (
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResponseWriter(t *testing.T) {
	t.Parallel()
	req := NewRequest(nil, "GET", "/abc")

	// Using NewResponse, vanilla
	r := NewResponse(req)
	r.Body = ioutil.NopCloser(strings.NewReader("boop"))
	assert.Equal(t, []byte("boop"), r.BodyBytes())

	// Using NewResponse, via ResponseWriter
	r = NewResponse(req)
	r.Writer().Header().Set("abc", "def")
	r.Writer().WriteHeader(http.StatusForbidden) // Test some other fun stuff while we're here
	r.Writer().Write([]byte("boop"))
	assert.Equal(t, []byte("boop"), r.BodyBytes())
	assert.Equal(t, http.StatusForbidden, r.StatusCode)
	assert.Equal(t, "def", r.Header.Get("abc"))

	// Using NewResponse, vanilla and then ResponseWriter
	r = NewResponse(req)
	r.Body = ioutil.NopCloser(strings.NewReader("boop"))
	r.Writer().Write([]byte("woop"))
	assert.Equal(t, []byte("boopwoop"), r.BodyBytes())
}

func TestResponseWriter_Error(t *testing.T) {
	t.Parallel()
	req := NewRequest(nil, "GET", "/")
	rsp := NewResponse(req)
	rsp.Writer().WriteError(errors.New("abc"))
	assert.Error(t, rsp.Error)
	assert.Equal(t, "abc", rsp.Error.Error())
}
