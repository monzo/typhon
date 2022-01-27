package typhon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	legacyproto "github.com/golang/protobuf/proto"
	"github.com/monzo/terrors"
	"google.golang.org/protobuf/proto"
)

// A Request is Typhon's wrapper around http.Request, used by both clients and servers.
//
// Note that Typhon makes no guarantees that a Request is safe to access or mutate concurrently. If a single Request
// object is to be used by multiple goroutines concurrently, callers must make sure to properly synchronise accesses.
type Request struct {
	http.Request
	context.Context
	err      error // Any error from request construction; read by ErrorFilter
	hijacker http.Hijacker
	server   *Server
}

// unwrappedContext returns the most "unwrapped" Context possible for that in the request.
// This is useful as it's very often the case that Typhon users will use a parent request
// as a parent for a child request. The context library knows how to unwrap its own
// types to most efficiently perform certain operations (eg. cancellation chaining), but
// it can't do that with Typhon-wrapped contexts.
func (r *Request) unwrappedContext() context.Context {
	switch c := r.Context.(type) {
	case Request:
		return c.unwrappedContext()
	case *Request:
		return c.unwrappedContext()
	default:
		return c
	}
}

// Encode maps to to EncodeAsJSON
func (r *Request) Encode(v interface{}) {
	r.EncodeAsJSON(v)
}

// EncodeAsJSON serialises the passed object as JSON into the body (and sets appropriate headers).
func (r *Request) EncodeAsJSON(v interface{}) {
	// If we were given an io.ReadCloser or an io.Reader (that is not also a json.Marshaler), use it directly
	switch v := v.(type) {
	case json.Marshaler:
	case io.ReadCloser:
		r.Body = v
		r.ContentLength = -1
		return
	case io.Reader:
		r.Body = ioutil.NopCloser(v)
		r.ContentLength = -1
		return
	}

	if err := json.NewEncoder(r).Encode(v); err != nil {
		r.err = terrors.Wrap(err, nil)
		return
	}
	r.Header.Set("Content-Type", "application/json")
}

// EncodeAsProtobuf serialises the passed object as protobuf into the body (and sets appropriate headers).
func (r *Request) EncodeAsProtobuf(m proto.Message) {
	out, err := proto.Marshal(m)
	if err != nil {
		r.err = terrors.Wrap(err, nil)
		return
	}

	n, err := r.Write(out)
	if err != nil {
		r.err = terrors.Wrap(err, nil)
		return
	}
	r.Header.Set("Content-Type", "application/protobuf")
	r.ContentLength = int64(n)
}

// Decode de-serialises the body into the passed object.
func (r Request) Decode(v interface{}) error {
	b, err := r.BodyBytes(true)
	if err != nil {
		return terrors.WrapWithCode(err, nil, terrors.ErrBadRequest)
	}

	switch r.Header.Get("Content-Type") {
	// application/x-protobuf is the "canonical" use, application/protobuf is defined in an expired IETF draft.
	// See: https://datatracker.ietf.org/doc/html/draft-rfernando-protocol-buffers-00#section-3.2
	// See: https://github.com/google/protorpc/blob/eb03145/python/protorpc/protobuf.py#L49-L51
	case "application/octet-stream", "application/x-google-protobuf", "application/protobuf", "application/x-protobuf":
		switch m := v.(type) {
		case proto.Message:
			err = proto.Unmarshal(b, m)
		case legacyproto.Message:
			err = legacyproto.Unmarshal(b, m)
		default:
			return terrors.InternalService("invalid_type", "could not decode proto message", nil)
		}
	// As older versions of typhon used json, we don't use protojson here as they are mutually exclusive standards with
	// major differences in how they handle some types (such as Enums)
	default:
		err = json.Unmarshal(b, v)
	}

	return terrors.WrapWithCode(err, nil, terrors.ErrBadRequest)
}

// Write writes the passed bytes to the request's body.
func (r *Request) Write(b []byte) (n int, err error) {
	switch rc := r.Body.(type) {
	// In the "normal" case, the response body will be a buffer, to which we can write
	case io.Writer:
		n, err = rc.Write(b)
		if err != nil {
			return n, err
		}
	// If a caller manually sets Response.Body, then we may not be able to write to it. In that case, we need to be
	// cleverer.
	default:
		buf := &bufCloser{}
		if rc != nil {
			if _, err := io.Copy(buf, rc); err != nil {
				// This can be quite bad; we have consumed (and possibly lost) some of the original body
				return 0, err
			}
			// rc will never again be accessible: once it's copied it must be closed
			rc.Close()
		}
		r.Body = buf
		n, err = buf.Write(b)
		if err != nil {
			return n, err
		}
	}

	if r.ContentLength >= 0 {
		r.ContentLength += int64(n)
		// If this write pushed the content length above the chunking threshold,
		// set to -1 (unknown) to trigger chunked encoding
		if r.ContentLength >= chunkThreshold {
			r.ContentLength = -1
		}
	}
	return n, nil
}

// BodyBytes fully reads the request body and returns the bytes read.
//
// If consume is true, this is equivalent to ioutil.ReadAll; if false, the caller will observe the body to be in
// the same state that it was before (ie. any remaining unread body can be read again).
func (r *Request) BodyBytes(consume bool) ([]byte, error) {
	if consume {
		defer r.Body.Close()
		return ioutil.ReadAll(r.Body)
	}

	switch rc := r.Body.(type) {
	case *bufCloser:
		return rc.Bytes(), nil
	default:
		buf := &bufCloser{}
		r.Body = buf
		rdr := io.TeeReader(rc, buf)
		// rc will never again be accessible: once it's copied it must be closed
		defer rc.Close()
		return ioutil.ReadAll(rdr)
	}
}

// Send round-trips the request via the default Client. It does not block, instead returning a ResponseFuture
// representing the asynchronous operation to produce the response. It is equivalent to:
//
//  r.SendVia(Client)
func (r Request) Send() *ResponseFuture {
	return Send(r)
}

// SendVia round-trips the request via the passed Service. It does not block, instead returning a ResponseFuture
// representing the asynchronous operation to produce the response.
func (r Request) SendVia(svc Service) *ResponseFuture {
	return SendVia(r, svc)
}

// Response constructs a new Response to the request, and if non-nil, encodes the given body into it.
func (r Request) Response(body interface{}) Response {
	rsp := NewResponse(r)
	if body != nil {
		rsp.Encode(body)
	}
	return rsp
}

// ResponseWithCode constructs a new Response with the given status code to the request, and if non-nil, encodes the
// given body into it.
func (r Request) ResponseWithCode(body interface{}, statusCode int) Response {
	rsp := NewResponseWithCode(r, statusCode)
	if body != nil {
		rsp.Encode(body)
	}
	return rsp
}

func (r Request) String() string {
	if r.URL == nil {
		return "Request(Unknown)"
	}
	return fmt.Sprintf("Request(%s %s://%s%s)", r.Method, r.URL.Scheme, r.Host, r.URL.Path)
}

// NewRequest constructs a new Request with the given parameters, and if non-nil, encodes the given body into it.
func NewRequest(ctx context.Context, method, url string, body interface{}) Request {
	if ctx == nil {
		ctx = context.Background()
	}
	httpReq, err := http.NewRequest(method, url, nil)
	req := Request{
		Context: ctx,
		err:     err}
	if httpReq != nil {
		httpReq.ContentLength = 0
		httpReq.Body = &bufCloser{}
		req.Request = *httpReq

		// Attach any metadata in the context to the request as headers.
		meta := MetadataFromContext(ctx)
		for k, v := range meta {
			req.Header[strings.ToLower(k)] = v
		}
	}
	if body != nil && err == nil {
		req.EncodeAsJSON(body)
	}
	return req
}
