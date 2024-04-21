package typhon

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"strings"
	"testing"

	"github.com/monzo/terrors"

	legacyproto "github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/monzo/typhon/legacyprototest"
	"github.com/monzo/typhon/prototest"
)

// TestRequestDecodeCloses verifies that a request body is closed after calling Decode()
func TestRequestDecodeCloses(t *testing.T) {
	t.Parallel()
	req := NewRequest(nil, "GET", "/", nil)
	b := []byte("{\"a\":\"b\"}\n")
	r := newDoneReader(io.NopCloser(bytes.NewReader(b)), -1)
	req.Body = r

	bout := map[string]string{}
	req.Decode(&bout)
	select {
	case <-r.closed:
	default:
		assert.Fail(t, "response body was not closed after Decode()")
	}
}

func TestRequestDecodeJSONStruct(t *testing.T) {
	req := NewRequest(nil, "GET", "/", nil)
	b := []byte("{\"message\":\"Hello world!\"}\n")
	r := newDoneReader(io.NopCloser(bytes.NewReader(b)), -1)
	req.Body = r

	g := &struct {
		Message string `json:"message"`
	}{}
	err := req.Decode(g)
	assert.NoError(t, err)
	assert.Equal(t, "Hello world!", g.Message)
}

func TestRequestDecodeProto(t *testing.T) {
	generateRequest := func() Request {
		req := NewRequest(nil, "GET", "/", nil)
		b, _ := proto.Marshal(&prototest.Greeting{Message: "Hello world!"})
		r := newDoneReader(io.NopCloser(bytes.NewReader(b)), -1)
		req.Header.Set("Content-Type", "application/protobuf")
		req.Body = r
		return req
	}

	req1 := generateRequest()
	g1 := &prototest.Greeting{}
	err := req1.Decode(g1)
	assert.NoError(t, err)
	assert.Equal(t, "Hello world!", g1.Message)

	req2 := generateRequest()
	g2 := &legacyprototest.LegacyGreeting{}
	err = req2.Decode(g2)
	assert.NoError(t, err)
	assert.Equal(t, "Hello world!", g2.Message)
}

func TestRequestDecodeProtoMaskingAsJSON(t *testing.T) {
	req := NewRequest(nil, "GET", "/", nil)
	b := []byte("{\"message\":\"Hello world!\"}\n")
	r := newDoneReader(io.NopCloser(bytes.NewReader(b)), -1)
	req.Body = r

	g := &prototest.Greeting{}
	err := req.Decode(g)
	assert.NoError(t, err)
	assert.Equal(t, "Hello world!", g.Message)
}

func TestRequestDecodeLegacyProto(t *testing.T) {
	generateRequest := func() Request {
		req := NewRequest(nil, "GET", "/", nil)
		b, _ := legacyproto.Marshal(&legacyprototest.LegacyGreeting{Message: "Hello world!"})
		r := newDoneReader(io.NopCloser(bytes.NewReader(b)), -1)
		req.Header.Set("Content-Type", "application/protobuf")
		req.Body = r
		return req
	}

	req1 := generateRequest()
	g1 := &prototest.Greeting{}
	err := req1.Decode(g1)
	assert.NoError(t, err)
	assert.Equal(t, "Hello world!", g1.Message)

	req2 := generateRequest()
	g2 := &legacyprototest.LegacyGreeting{}
	err = req2.Decode(g2)
	assert.NoError(t, err)
	assert.Equal(t, "Hello world!", g2.Message)
}

func TestRequestDecodeLegacyProtoMaskingAsJSON(t *testing.T) {
	req := NewRequest(nil, "GET", "/", nil)
	b := []byte("{\"message\":\"Hello world!\"}\n")
	r := newDoneReader(io.NopCloser(bytes.NewReader(b)), -1)
	req.Body = r

	g := &legacyprototest.LegacyGreeting{}
	err := req.Decode(g)
	assert.NoError(t, err)
	assert.Equal(t, "Hello world!", g.Message)
}

func TestRequestDecodeErrorGivesTerror(t *testing.T) {
	req := NewRequest(nil, "GET", "/", nil)
	req.Body = io.NopCloser(strings.NewReader("invalid json"))

	bout := map[string]string{}
	err := req.Decode(&bout)
	assert.True(t, terrors.Is(err, "bad_request"))
	assert.True(t, terrors.Matches(err, "invalid character 'i'"))
}

// TestRequestEncodeReader verifies that passing an io.Reader to request.Encode() uses it properly as the body, and
// does not attempt to encode it as JSON
func TestRequestEncodeReader(t *testing.T) {
	t.Parallel()

	// io.ReadCloser: this should be used with no modification
	rc := io.NopCloser(strings.NewReader("hello world"))
	req := NewRequest(nil, "GET", "/", nil)
	req.Encode(rc)
	assert.Equal(t, req.Body, rc)
	assert.EqualValues(t, -1, req.ContentLength)
	assert.Empty(t, req.Header.Get("Content-Type"))

	// io.Reader: this should be wrapped in an io.NopCloser
	r := strings.NewReader("hello world, again")
	req = NewRequest(nil, "GET", "/", nil)
	req.Encode(r)
	assert.EqualValues(t, -1, req.ContentLength)
	assert.Empty(t, req.Header.Get("Content-Type"))
	body, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	assert.Equal(t, []byte("hello world, again"), body)

	// an io.ReadCloser that happens to implement json.Marshaler should not be used directly and should be marshaled
	jm := jsonMarshalerReader{
		ReadCloser: io.NopCloser(strings.NewReader("this should never see the light of day"))}
	req = NewRequest(nil, "GET", "/", nil)
	req.Encode(jm)
	assert.EqualValues(t, 3, req.ContentLength)
	assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
	body, err = io.ReadAll(req.Body)
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

	bodyBytes, err := io.ReadAll(req.Body)
	require.NoError(t, err)

	assert.Equal(t, "application/protobuf", req.Header.Get("Content-Type"))
	assert.EqualValues(t, bodyBytes, protoContentForComparison)
}

func TestRequestEncodeJSON(t *testing.T) {
	message := map[string]interface{}{
		"foo": "bar",
		"bar": 3,
	}

	jsonContentForComparison, err := jsonStreamMarshal(message)
	require.NoError(t, err)

	req := NewRequest(nil, "GET", "/", nil)
	req.Encode(message)

	bodyBytes, err := io.ReadAll(req.Body)
	require.NoError(t, err)

	assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
	assert.EqualValues(t, bodyBytes, jsonContentForComparison)
}

func TestRequestEncodeErrorGivesTerror(t *testing.T) {
	req := NewRequest(nil, "GET", "/", nil)
	req.Encode(math.Inf(1))
	assert.True(t, terrors.Is(req.err, "internal_service"))
	assert.True(t, terrors.Matches(req.err, "unsupported value"))
}

func TestRequestSetMetadata(t *testing.T) {
	t.Parallel()

	ctx := AppendMetadataToContext(context.Background(), NewMetadata(map[string]string{
		"meta": "data",
	}))

	req := NewRequest(ctx, "GET", "/", nil)

	assert.Equal(t, []string{"data"}, req.Request.Header["meta"])
}

func TestRouterEndpointPattern(t *testing.T) {
	req := NewRequest(context.Background(), http.MethodGet, "/foo/some-url-identifier", nil)
	assert.Equal(t, "", req.RequestPathPattern()) // should be empty if request has not been served by a router

	router := Router{}
	routerEndpointPattern := "/foo/:id"
	router.GET(routerEndpointPattern, func(req Request) Response {
		// as we are currently serving the request, we should be able to get the router endpoint pattern
		assert.Equal(t, routerEndpointPattern, req.RequestPathPattern())
		return req.Response(nil)
	})

	rsp := req.SendVia(router.Serve()).Response()
	require.NoError(t, rsp.Error) // check we didn't get a "route not found" error
}

func TestRequestMethod(t *testing.T) {
	req := NewRequest(context.Background(), http.MethodGet, "", nil)
	assert.Equal(t, http.MethodGet, req.RequestMethod())
}

func jsonStreamMarshal(v interface{}) ([]byte, error) {
	var buffer bytes.Buffer
	writer := bufio.NewWriter(&buffer)

	err := json.NewEncoder(writer).Encode(v)
	if err != nil {
		return nil, err
	}

	err = writer.Flush()
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}
