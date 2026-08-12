package typhon

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/monzo/typhon/prototest"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

// TestHttpServiceSetsGetBody verifies that HttpService populates Request.GetBody for buffered bodies,
// so the HTTP/2 transport can rewind and transparently retry the request after a mid-flight connection
// teardown (eg. a graceful-shutdown GOAWAY).
func TestHttpServiceSetsGetBody(t *testing.T) {
	t.Parallel()

	req := NewRequest(context.Background(), "POST", "/", nil)
	req.EncodeAsProtobuf(&prototest.Greeting{Message: "Hello world!"})
	want, err := req.BodyBytes(false)
	require.NoError(t, err)

	var captured *http.Request
	svc := HttpService(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		captured = r
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(nil))}, nil
	}))
	rsp := svc(req)
	require.NoError(t, rsp.Error)

	require.NotNil(t, captured.GetBody, "GetBody should be set for a buffered body")
	rc, err := captured.GetBody()
	require.NoError(t, err)
	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// TestHttpServiceStreamingBodyHasNoGetBody verifies that streaming (non-buffered) bodies remain
// non-replayable - we must not claim a stream can be rewound.
func TestHttpServiceStreamingBodyHasNoGetBody(t *testing.T) {
	t.Parallel()

	req := NewRequest(context.Background(), "POST", "/", nil)
	req.Encode(io.NopCloser(bytes.NewReader([]byte("streamed"))))

	var captured *http.Request
	svc := HttpService(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		captured = r
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(nil))}, nil
	}))
	rsp := svc(req)
	require.NoError(t, rsp.Error)

	assert.Nil(t, captured.GetBody, "GetBody must stay nil for a streaming body")
}
