package typhon

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	log "github.com/mondough/slog"
	"github.com/mondough/terrors"
	"golang.org/x/net/context"
)

type Response struct {
	*http.Response
	Error error
	ctx   context.Context
}

// Encode serialises the passed object as JSON into the body (and sets appropriate headers).
func (r *Response) Encode(v interface{}) {
	if err := json.NewEncoder(r).Encode(v); err != nil {
		r.Error = terrors.Wrap(err, nil)
		log.Warn(r.ctx, "Failed to encode response body: %v", err)
		return
	}
	r.Header.Set("Content-Type", "application/json")
}

// Decode de-serialises the JSON body into the passed object.
func (r *Response) Decode(v interface{}) error {
	err := error(nil)
	if r.Response == nil {
		err = terrors.InternalService("", "Response has no body", nil)
		log.Warn(r.ctx, "Cannot decode response with no Body (Response is nil)", nil)
	} else {
		err := json.NewDecoder(r.Body).Decode(v)
		if err != nil {
			log.Warn(r.ctx, "Failed to decode response body: %v", err)
		}
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
		}
		r.Body = buf
		return buf.Write(b)
	}
}

func (r *Response) BodyBytes(consume bool) ([]byte, error) {
	if consume {
		return ioutil.ReadAll(r.Body)
	}

	switch rc := r.Body.(type) {
	case *bufCloser:
		return rc.Bytes(), nil
	default:
		rdr := io.Reader(rc)
		buf := &bufCloser{}
		r.Body = buf
		rdr = io.TeeReader(rdr, buf)
		return ioutil.ReadAll(rdr)
	}
}

// Writer returns a ResponseWriter proxy.
func (r *Response) Writer() ResponseWriter {
	return responseWriterWrapper{
		r: r}
}

func (r *Response) String() string {
	if r != nil {
		if r.Response != nil {
			return fmt.Sprintf("Response(%d, error: %v)", r.StatusCode, r.Error)
		}
		return fmt.Sprintf("Response(???, error: %v)", r.Error)
	}
	return "Response(Unknown)"
}

func newHttpResponse(req Request) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK, // Seems like a reasonable default
		Proto:      req.Proto,
		ProtoMajor: req.ProtoMajor,
		ProtoMinor: req.ProtoMinor,
		Header:     make(http.Header, 5),
		Body:       &bufCloser{}}
}

// NewResponse constructs a Response
func NewResponse(req Request) Response {
	return Response{
		ctx:      req.Context,
		Error:    nil,
		Response: newHttpResponse(req)}
}
