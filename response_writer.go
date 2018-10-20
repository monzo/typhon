package typhon

import (
	"bufio"
	"net"
	"net/http"
)

type ResponseWriter interface {
	http.ResponseWriter
	// WriteJSON writes the given data as JSON to the Response. The passed value must (perhaps obviously) be
	// serialisable to JSON.
	WriteJSON(interface{})
	// WriteError writes the given error to the Response.
	WriteError(err error)
}

type responseWriterWrapper struct {
	r *Response
}

func (rw responseWriterWrapper) Header() http.Header {
	return rw.r.Header
}

func (rw responseWriterWrapper) Write(b []byte) (int, error) {
	return rw.r.Write(b)
}

func (rw responseWriterWrapper) WriteHeader(status int) {
	rw.r.StatusCode = status
}

func (rw responseWriterWrapper) WriteJSON(v interface{}) {
	rw.r.Encode(v)
}

func (rw responseWriterWrapper) WriteError(err error) {
	rw.r.Error = err
}

type hijackerRw struct {
	responseWriterWrapper
	http.Hijacker
}

func (rw hijackerRw) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	rw.r.hijacked = true
	return rw.Hijacker.Hijack()
}
