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

type Request struct {
	http.Request
	context.Context
}

// Encode serialises the passed object as JSON into the body (and sets appropriate headers).
func (r *Request) Encode(v interface{}) {
	if err := json.NewEncoder(r).Encode(v); err != nil {
		terr := terrors.Wrap(err, nil)
		log.Warn(r, "Failed to encode request body: %v", terr)
		return
	}
	r.Header.Set("Content-Type", "application/json")
}

// Decode de-serialises the JSON body into the passed object.
func (r Request) Decode(v interface{}) error {
	err := json.NewDecoder(r.Body).Decode(v)
	if err != nil {
		log.Warn(r, "Failed to decode response body: %v", err)
	}
	return terrors.WrapWithCode(err, nil, terrors.ErrBadRequest)
}

func (r *Request) Write(b []byte) (int, error) {
	switch rc := r.Body.(type) {
	// In the "regular" case, the response body will be a bufCloser; we can write
	case io.Writer:
		return rc.Write(b)
	// If a caller manually sets Response.Body, then we may not be able to write to it. In that case, we need to be
	// cleverer.
	default:
		buf := &bufCloser{}
		if _, err := io.Copy(buf, rc); err != nil {
			// This can be quite bad; we have consumed (and possibly lost) some of the original body
			return 0, err
		}
		r.Body = buf
		return buf.Write(b)
	}
}

func (r *Request) BodyBytes(consume bool) ([]byte, error) {
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

func (r Request) Send() *ResponseFuture {
	return Send(r)
}

func (r Request) SendVia(svc Service) *ResponseFuture {
	return SendVia(r, svc)
}

func (r Request) Response(body interface{}) Response {
	rsp := NewResponse(r)
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

func NewRequest(ctx context.Context, method, url string, body interface{}) Request {
	if ctx == nil {
		ctx = context.Background()
	}
	httpReq, _ := http.NewRequest(method, url, nil) // @TODO: Don't swallow this error
	httpReq.Body = &bufCloser{}
	req := Request{
		Request: *httpReq,
		Context: ctx}
	if body != nil {
		req.Encode(body)
	}
	return req
}
