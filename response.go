package typhon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/monzo/terrors"
)

type Response struct {
	*http.Response
	Error error
	ctx   context.Context
}

// Encode serialises the passed object as JSON into the body (and sets appropriate headers).
func (r *Response) Encode(v interface{}) {
	cw := &countingWriter{
		Writer: r}
	if err := json.NewEncoder(cw).Encode(v); err != nil {
		r.Error = terrors.Wrap(err, nil)
		return
	}
	r.Header.Set("Content-Type", "application/json")
	if r.ContentLength < 0 {
		r.ContentLength = int64(cw.n)
	}
}

// Decode de-serialises the JSON body into the passed object.
func (r *Response) Decode(v interface{}) error {
	err := error(nil)
	if r.Error != nil {
		return r.Error
	} else if r.Response == nil {
		err = terrors.InternalService("", "Response has no body", nil)
	} else {
		defer r.Body.Close()
		err = json.NewDecoder(r.Body).Decode(v)
		err = terrors.WrapWithCode(err, nil, terrors.ErrBadResponse)
	}
	if r.Error == nil {
		r.Error = err
	}
	return err
}

func (r *Response) Write(b []byte) (int, error) {
	if r.Response == nil {
		r.Response = newHttpResponse(Request{})
	}
	switch rc := r.Body.(type) {
	// In the "regular" case, the response body will be a bufCloser; we can write
	case io.Writer:
		return rc.Write(b)
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
		return buf.Write(b)
	}
}

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

// Writer returns a ResponseWriter proxy.
func (r *Response) Writer() ResponseWriter {
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

func newHttpResponse(req Request) *http.Response {
	return &http.Response{
		StatusCode:    http.StatusOK, // Seems like a reasonable default
		Proto:         req.Proto,
		ProtoMajor:    req.ProtoMajor,
		ProtoMinor:    req.ProtoMinor,
		ContentLength: -1,
		Header:        make(http.Header, 5),
		Body:          &bufCloser{}}
}

// NewResponse constructs a Response
func NewResponse(req Request) Response {
	return Response{
		ctx:      req.Context,
		Error:    nil,
		Response: newHttpResponse(req)}
}
