package typhon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/monzo/terrors"
)

// A Response is Typhon's wrapper around http.Response, used by both clients and servers.
//
// Note that Typhon makes no guarantees that a Response is safe to access or mutate concurrently. If a single Response
// object is to be used by multiple goroutines concurrently, callers must make sure to properly synchronise accesses.
type Response struct {
	*http.Response
	Error    error
	Request  *Request // The Request that we are responding to
	hijacked bool
}

// Encode serialises the passed object as JSON into the body (and sets appropriate headers).
func (r *Response) Encode(v interface{}) {
	if r.Response == nil {
		r.Response = newHTTPResponse(Request{})
	}

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
		r.Error = terrors.Wrap(err, nil)
		return
	}
	r.Header.Set("Content-Type", "application/json")
}

// Decode de-serialises the JSON body into the passed object.
func (r *Response) Decode(v interface{}) error {
	if r.Error != nil {
		return r.Error
	}
	err := error(nil)
	if r.Response == nil {
		err = terrors.InternalService("", "Response has no body", nil)
	} else {
		var b []byte
		b, err = r.BodyBytes(true)
		if err == nil {
			err = json.Unmarshal(b, v)
		}
		err = terrors.WrapWithCode(err, nil, terrors.ErrBadResponse)
	}
	if r.Error == nil {
		r.Error = err
	}
	return err
}

// Write writes the passed bytes to the response's body.
func (r *Response) Write(b []byte) (n int, err error) {
	if r.Response == nil {
		r.Response = newHTTPResponse(Request{})
	}
	switch rc := r.Body.(type) {
	// In the "regular" case, the response body will be a bufCloser; we can write
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

// BodyBytes fully reads the response body and returns the bytes read. If consume is false, the body is copied into a
// new buffer such that it may be read again.
func (r *Response) BodyBytes(consume bool) ([]byte, error) {
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

// Writer returns a ResponseWriter which can be used to populate the response.
//
// This is useful when you want to use another HTTP library that is used to wrapping net/http directly. For example,
// it allows a Typhon Service to use a http.Handler internally.
func (r *Response) Writer() ResponseWriter {
	if r.Request != nil && r.Request.hijacker != nil {
		return hijackerRw{
			responseWriterWrapper: responseWriterWrapper{
				r: r},
			Hijacker: r.Request.hijacker}
	}
	return responseWriterWrapper{
		r: r}
}

func (r Response) String() string {
	b := new(bytes.Buffer)
	fmt.Fprint(b, "Response(")
	if r.Response != nil {
		fmt.Fprintf(b, "%d", r.StatusCode)
	} else {
		fmt.Fprint(b, "???")
	}
	if r.Error != nil {
		fmt.Fprintf(b, ", error: %v", r.Error)
	}
	fmt.Fprint(b, ")")
	return b.String()
}

func newHTTPResponse(req Request) *http.Response {
	return &http.Response{
		StatusCode:    http.StatusOK, // Seems like a reasonable default
		Proto:         req.Proto,
		ProtoMajor:    req.ProtoMajor,
		ProtoMinor:    req.ProtoMinor,
		ContentLength: 0,
		Header:        make(http.Header, 5),
		Body:          &bufCloser{}}
}

// NewResponse constructs a Response
func NewResponse(req Request) Response {
	return Response{
		Request:  &req,
		Error:    nil,
		Response: newHTTPResponse(req)}
}
