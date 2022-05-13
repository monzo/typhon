package typhon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"strings"
	"testing"

	legacyproto "github.com/golang/protobuf/proto"
	"github.com/monzo/typhon/legacyprototest"

	"github.com/monzo/terrors"
	"github.com/monzo/typhon/prototest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestResponseWriter(t *testing.T) {
	t.Parallel()
	req := Request{}

	// Using NewResponse, vanilla
	r := NewResponse(req)
	r.Write([]byte("boop"))
	b, _ := r.BodyBytes(true)
	assert.Equal(t, []byte("boop"), b)
	assert.EqualValues(t, 4, r.ContentLength)
	assert.Equal(t, http.StatusOK, r.StatusCode)

	// Using NewResponse, via ResponseWriter
	r = NewResponse(req)
	r.Writer().Header().Set("abc", "def")
	r.Writer().WriteHeader(http.StatusForbidden) // Test some other fun stuff while we're here
	r.Writer().Write([]byte("boop"))
	b, _ = r.BodyBytes(true)
	assert.Equal(t, []byte("boop"), b)
	assert.EqualValues(t, 4, r.ContentLength)
	assert.Equal(t, http.StatusForbidden, r.StatusCode)
	assert.Equal(t, "def", r.Header.Get("abc"))

	// Using NewResponse, vanilla and then ResponseWriter
	r = NewResponse(req)
	r.Write([]byte("boop"))
	r.Writer().Write([]byte("woop"))
	b, _ = r.BodyBytes(true)
	assert.Equal(t, []byte("boopwoop"), b)
	assert.EqualValues(t, 8, r.ContentLength)
	assert.Equal(t, http.StatusOK, r.StatusCode)
}

func TestResponseWithCodeWriter(t *testing.T) {
	t.Parallel()
	req := Request{}

	// Using NewResponseWithCode, vanilla
	r := NewResponseWithCode(req, http.StatusCreated)
	r.Write([]byte("boop"))
	b, _ := r.BodyBytes(true)
	assert.Equal(t, []byte("boop"), b)
	assert.EqualValues(t, 4, r.ContentLength)
	assert.Equal(t, http.StatusCreated, r.StatusCode)

	// Using NewResponseWithCode, via ResponseWriter
	r = NewResponseWithCode(req, http.StatusCreated)
	r.Writer().Header().Set("abc", "def")
	r.Writer().Write([]byte("boop"))
	b, _ = r.BodyBytes(true)
	assert.Equal(t, []byte("boop"), b)
	assert.EqualValues(t, 4, r.ContentLength)
	assert.Equal(t, http.StatusCreated, r.StatusCode)
	assert.Equal(t, "def", r.Header.Get("abc"))

	// Using NewResponseWithCode, vanilla and then ResponseWriter
	r = NewResponseWithCode(req, http.StatusCreated)
	r.Write([]byte("boop"))
	r.Writer().Write([]byte("woop"))
	b, _ = r.BodyBytes(true)
	assert.Equal(t, []byte("boopwoop"), b)
	assert.EqualValues(t, 8, r.ContentLength)
	assert.Equal(t, http.StatusCreated, r.StatusCode)
}

func TestResponseWriter_Error(t *testing.T) {
	t.Parallel()
	rsp := NewResponse(Request{})
	rsp.Writer().WriteError(errors.New("abc"))
	assert.Error(t, rsp.Error)
	assert.Equal(t, "abc", rsp.Error.Error())
}

func TestResponse_WrapDownstreamErrors(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		desc            string
		downstreamErr   error
		ctx             context.Context
		expectedErrCode string
	}{
		{
			desc:            "missing context",
			ctx:             nil,
			downstreamErr:   terrors.NotFound("foo", "not found", nil),
			expectedErrCode: "not_found.foo",
		},
		{
			desc:            "wrap downstream errors not set",
			ctx:             context.Background(),
			downstreamErr:   terrors.NotFound("foo", "not found", nil),
			expectedErrCode: "not_found.foo",
		},
		{
			desc:            "wrap downstream errors set",
			ctx:             context.WithValue(context.Background(), WrapDownstreamErrors{}, "1"),
			downstreamErr:   terrors.NotFound("foo", "not found", nil),
			expectedErrCode: "internal_service.downstream",
		},
		{
			desc:            "wrap downstream errors empty",
			ctx:             context.WithValue(context.Background(), WrapDownstreamErrors{}, ""),
			downstreamErr:   terrors.NotFound("foo", "not found", nil),
			expectedErrCode: "not_found.foo",
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			req := Request{}
			req.Context = tc.ctx
			rsp := NewResponse(req)
			rsp.Error = tc.downstreamErr
			err := rsp.Decode(nil)
			require.Error(t, err)
			assert.Truef(t, terrors.PrefixMatches(err, tc.expectedErrCode),
				"expected error with code: [%s] got: [%s]", tc.expectedErrCode, err.Error(),
			)
		})
	}
}

func TestResponse_WrapDownstreamErrorsWithoutRequest(t *testing.T) {
	t.Parallel()

	// It's possible to create a Response without a Request.
	// This test ensures that we don't panic when trying to read from the context.
	err := terrors.NotFound("foo", "not found", nil)
	rsp := Response{Error: err}
	assert.Equal(t, err, rsp.Decode(nil))
}

// TestResponseDecodeCloses verifies that a response body is closed after calling Decode()
func TestResponseDecodeCloses(t *testing.T) {
	t.Parallel()
	rsp := NewResponse(Request{})
	b := []byte(`{"a":"b"}` + "\n")
	r := newDoneReader(ioutil.NopCloser(bytes.NewReader(b)), -1)
	rsp.Body = r

	bout := map[string]string{}
	assert.NoError(t, rsp.Decode(&bout))
	assert.Equal(t, map[string]string{
		"a": "b"}, bout)
	select {
	case <-r.closed:
	default:
		assert.Fail(t, "response body was not closed after Decode()")
	}
}

// TestResponseDecodeJSON_TrailingSpace verifies that trailing newlines do not result in a decoding error
func TestResponseDecodeJSON_TrailingSpace(t *testing.T) {
	t.Parallel()
	rsp := NewResponse(Request{})
	rsp.Body = ioutil.NopCloser(strings.NewReader(`{"foo":"bar"}` + "\n\n\n\n"))

	bout := map[string]string{}
	assert.NoError(t, rsp.Decode(&bout))
	assert.Equal(t, map[string]string{
		"foo": "bar"}, bout)
}

// TestResponseDecodeProtobuf verifies decoding of a protobuf message
func TestResponseDecodeProtobuf(t *testing.T) {
	t.Parallel()

	g := &prototest.Greeting{
		Message:  "Hello world!",
		Priority: 1}
	b, _ := proto.Marshal(g)
	rsp := NewResponse(Request{})
	rsp.Body = ioutil.NopCloser(bytes.NewReader(b))
	rsp.Header.Set("Content-Type", "application/protobuf")

	gout := &prototest.Greeting{}
	assert.NoError(t, rsp.Decode(gout))
	assert.Equal(t, "Hello world!", gout.Message)
	assert.EqualValues(t, 1, gout.Priority)
}

// TestResponseDecodeProtobufWithAltType verifies decoding of a protobuf message
func TestResponseDecodeProtobufWithAltType(t *testing.T) {
	t.Parallel()

	g := &prototest.Greeting{
		Message:  "Hello world!",
		Priority: 1}
	b, _ := proto.Marshal(g)
	rsp := NewResponse(Request{})
	rsp.Body = ioutil.NopCloser(bytes.NewReader(b))
	rsp.Header.Set("Content-Type", "application/x-protobuf")

	gout := &prototest.Greeting{}
	assert.NoError(t, rsp.Decode(gout))
	assert.Equal(t, "Hello world!", gout.Message)
	assert.EqualValues(t, 1, gout.Priority)
}

// TestResponseDecodeLegacyProtobuf verifies decoding of a legacy protobuf message
func TestResponseDecodeLegacyProtobuf(t *testing.T) {
	t.Parallel()

	g := &legacyprototest.LegacyGreeting{
		Message:  "Hello world!",
		Priority: 1,
	}
	b, _ := legacyproto.Marshal(g)
	rsp := NewResponse(Request{})
	rsp.Body = ioutil.NopCloser(bytes.NewReader(b))
	rsp.Header.Set("Content-Type", "application/protobuf")

	gout := &legacyprototest.LegacyGreeting{}
	assert.NoError(t, rsp.Decode(gout))
	assert.Equal(t, "Hello world!", gout.Message)
	assert.EqualValues(t, 1, gout.Priority)
}

// TestResponseDecodeLegacyProtobufWithAltType verifies decoding of a legacy protobuf message
func TestResponseDecodeLegacyProtobufWithAltType(t *testing.T) {
	t.Parallel()

	g := &legacyprototest.LegacyGreeting{
		Message:  "Hello world!",
		Priority: 1,
	}
	b, _ := legacyproto.Marshal(g)
	rsp := NewResponse(Request{})
	rsp.Body = ioutil.NopCloser(bytes.NewReader(b))
	rsp.Header.Set("Content-Type", "application/x-protobuf")

	gout := &prototest.Greeting{}
	assert.NoError(t, rsp.Decode(gout))
	assert.Equal(t, "Hello world!", gout.Message)
	assert.EqualValues(t, 1, gout.Priority)
}

func TestResponseDecodeErrorGivesTerror(t *testing.T) {
	rsp := NewResponse(Request{})
	rsp.Body = ioutil.NopCloser(strings.NewReader("invalid json"))

	bout := map[string]string{}
	err := rsp.Decode(&bout)
	assert.True(t, terrors.Is(err, "bad_response"))
	assert.True(t, terrors.Matches(err, "invalid character 'i'"))
}

// rc is a helper type used in tests involving a generic io.ReadCloser
type rc struct {
	strings.Reader
	closed int
}

func (v *rc) Close() error {
	v.closed += 1
	return nil
}

// TestResponseBodyBytes_Consuming verifies that Response.BodyBytes behaves as expected in consuming mode (ie. where it
// is expected that future calls to BodyBytes() will return EOF).
//
// The BodyBytes function is specialised for efficiency on some common types that Typhon uses as a Response.Body; this
// test verifies that these specialisations work as expected along with the general io.ReadCloser case.
func TestResponseBodyBytes_Consuming(t *testing.T) {
	t.Parallel()

	// Most general case: an opaque io.ReadCloser
	body := &rc{*strings.NewReader("abc"), 0}
	rsp := NewResponse(Request{})
	rsp.Body = body
	b, err := rsp.BodyBytes(true)
	require.NoError(t, err)
	assert.Equal(t, []byte("abc"), b)
	assert.Equal(t, 1, body.closed) // The reader should have been closed

	// Specialised case: *bufCloser
	rsp.Body = &bufCloser{*bytes.NewBuffer([]byte("def"))}
	b, err = rsp.BodyBytes(true)
	require.NoError(t, err)
	assert.Equal(t, []byte("def"), b)
}

// TestResponseBodyBytes_Preserving verifies that Response.BodyBytes behaves as expected in consuming mode (ie. where it
// is expected that future calls to BodyBytes() will return EOF).
//
// The BodyBytes function is specialised for efficiency on some common types that Typhon uses as a Response.Body; this
// test verifies that these specialisations work as expected along with the general io.ReadCloser case.
func TestResponseBodyBytes_Preserving(t *testing.T) {
	t.Parallel()

	// Most general case: an opaque io.ReadCloser
	body := &rc{*strings.NewReader("abc"), 0}
	rsp := NewResponse(Request{})
	rsp.Body = body
	for i := 0; i < 10; i++ { // Repeated reads should yield the same result
		b, err := rsp.BodyBytes(false)
		require.NoError(t, err)
		assert.Equal(t, []byte("abc"), b)
		assert.Equal(t, 1, body.closed) // The underlying reader should have been closed exactly once
	}

	// Specialised case: *bufCloser
	rsp.Body = &bufCloser{*bytes.NewBuffer([]byte("def"))}
	for i := 0; i < 100; i++ { // Repeated reads should yield the same result
		b, err := rsp.BodyBytes(false)
		require.NoError(t, err)
		assert.Equal(t, []byte("def"), b)
	}
}

type jsonMarshalerReader struct {
	io.ReadCloser
}

func (r jsonMarshalerReader) MarshalJSON() ([]byte, error) {
	return []byte("{}"), nil
}

type protoMarshalerReader struct {
	io.ReadCloser
	*prototest.Greeting
}

// TestResponseEncodeReader verifies that passing an io.Reader to response.Encode() uses it properly as the body, and
// does not attempt to encode it as JSON or Protobuf
func TestResponseEncodeReader(t *testing.T) {
	t.Parallel()

	// io.ReadCloser: this should be used with no modification
	rc := ioutil.NopCloser(strings.NewReader("hello world"))
	rsp := Response{}
	rsp.Encode(rc)
	assert.Nil(t, rsp.Error)
	assert.Equal(t, rsp.Body, rc)
	assert.EqualValues(t, -1, rsp.ContentLength)
	assert.Empty(t, rsp.Header.Get("Content-Type"))

	// io.Reader: this should be wrapped in an ioutil.NopCloser
	r := strings.NewReader("hello world, again")
	rsp = Response{}
	rsp.Encode(r)
	assert.Nil(t, rsp.Error)
	assert.EqualValues(t, -1, rsp.ContentLength)
	assert.Empty(t, rsp.Header.Get("Content-Type"))
	body, err := ioutil.ReadAll(rsp.Body)
	require.NoError(t, err)
	assert.Equal(t, []byte("hello world, again"), body)

	// an io.ReadCloser that happens to implement json.Marshaler should not be used directly and should be marshaled
	jm := jsonMarshalerReader{
		ReadCloser: ioutil.NopCloser(strings.NewReader("this should never see the light of day"))}
	assert.Implements(t, (*json.Marshaler)(nil), jm)
	rsp = Response{}
	rsp.Encode(jm)
	assert.Nil(t, rsp.Error)
	assert.EqualValues(t, 3, rsp.ContentLength)
	assert.Equal(t, "application/json", rsp.Header.Get("Content-Type"))
	body, err = ioutil.ReadAll(rsp.Body)
	require.NoError(t, err)
	assert.Equal(t, []byte("{}\n"), body)

	// an io.ReadCloser that implements proto.Message should be marshaled
	pm := &protoMarshalerReader{
		ReadCloser: ioutil.NopCloser(strings.NewReader("this should never see the light of day")),
		Greeting: &prototest.Greeting{
			Message: "hello",
		},
	}
	assert.Implements(t, (*proto.Message)(nil), pm)
	req := NewRequest(nil, "GET", "/", nil)
	req.Header.Set("Accept", "application/protobuf")
	rsp = Response{Request: &req}
	rsp.Encode(pm)
	assert.Nil(t, rsp.Error)
	assert.Equal(t, "application/protobuf", rsp.Header.Get("Content-Type"))
	body, err = ioutil.ReadAll(rsp.Body)
	require.NoError(t, err)
	assert.Subset(t, body, []byte("hello"), "'hello' should appear in the wire format")
}

func TestResponseEncodeErrorGivesTerror(t *testing.T) {
	rsp := NewResponse(Request{})
	rsp.Encode(math.Inf(1))
	assert.True(t, terrors.Is(rsp.Error, "internal_service"))
	assert.True(t, terrors.Matches(rsp.Error, "unsupported value"))
}
