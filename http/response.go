package httpsvc

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"

	log "github.com/mondough/slog"
	"golang.org/x/net/context"
)

type Response struct {
	*http.Response
	Error   error
	Context context.Context
}

// SetBody is a convenience for setting the entire body content at once.
func (r Response) SetBody(b []byte) {
	r.Body = ioutil.NopCloser(bytes.NewReader(b))
}

// BodyBytes consumes all of the Respnse body and returns it as a byte slice
func (r Response) BodyBytes() []byte {
	if r.Body == nil {
		return nil
	}
	b, err := ioutil.ReadAll(r.Body)
	if err == nil {
		err = r.Body.Close()
	}
	if err != nil {
		log.Error(r.Context, "Could not read response body: %v", err)
		return nil
	}
	return b
}

// WriteTo writes the response out to the given http.ResponseWriter.
// Streaming bodies are supported.
func (r Response) WriteTo(ctx context.Context, rw http.ResponseWriter) {
	h := rw.Header()
	for k, v := range r.Header {
		h[k] = v
	}
	rw.WriteHeader(r.StatusCode)
	if r.Body != nil {
		defer r.Body.Close()
		if _, err := io.Copy(rw, r.Body); err != nil {
			log.Error(ctx, "[Typhon:http:networkTransport] Error copying response body: %v", err)
		}
	}
}

// Writer returns a ResponseWriter proxy.
func (r *Response) Writer() ResponseWriter {
	return responseWriterWrapper{
		r: r}
}

// NewResponse constructs a Response
func NewResponse(req Request) Response {
	return Response{
		Context: req,
		Error:   nil,
		Response: &http.Response{
			StatusCode: http.StatusOK, // Seems like a reasonable default
			Proto:      req.Proto,
			ProtoMajor: req.ProtoMajor,
			ProtoMinor: req.ProtoMinor,
			Header:     make(http.Header, 5),
			Body:       &bufCloser{}}}
}
