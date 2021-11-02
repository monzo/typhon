package typhon

import (
	"io"
	"net"
	"net/http"
	"net/textproto"
	"os"
	"strings"
	"sync"
	"syscall"

	"github.com/monzo/slog"
	"golang.org/x/net/http/httpguts"
)

const (
	// chunkThreshold is a byte threshold above which request and response bodies that result from using buffered I/O
	// within Typhon will be transferred with chunked encoding on the wire.
	chunkThreshold = 5 * 1000000 // 5 megabytes
)

var httpChunkBufPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 32*1024) // size is the same as io.Copy uses internally
		return &buf
	}}

func isStreamingRsp(rsp Response) bool {
	// Most straightforward: service may have set rsp.Body to a streamer
	if s, ok := rsp.Body.(*streamer); ok && s != nil {
		return true
	}

	// If the content length is unknown, it should stream
	if rsp.ContentLength <= 0 {
		return true
	}

	// If the response body is the same as the request body and the request is streaming, the response should be too
	if rsp.Request != nil && rsp.Request.ContentLength <= 0 && rsp.Body == rsp.Request.Body {
		return true
	}

	// Chunked transfer encoding (only in HTTP/1.1) gives us an additional clue
	if !rsp.ProtoAtLeast(2, 0) {
		if httpguts.HeaderValuesContainsToken(rsp.Header[textproto.CanonicalMIMEHeaderKey("Transfer-Encoding")], "chunked") {
			return true
		}
		// Annoyingly, this can be removed from headers by net/http and promoted to its own field
		for _, v := range rsp.TransferEncoding {
			if v == "chunked" {
				return true
			}
		}
	}

	return false
}

// copyErrSeverity returns a slog error severity that should be used to report an error from an io.Copy operation to
// send the response body to a client. This exists because these errors often do not indicate actual problems. For
// example, a client may disconnect before the response body is copied to it; this doesn't mean the server is
// misbehaving.
func copyErrSeverity(err error) slog.Severity {

	switch {
	case strings.HasSuffix(err.Error(), "read on closed response body"),
		strings.HasSuffix(err.Error(), "connection reset by peer"):
		return slog.DebugSeverity
	}

	// Annoyingly, these errors can be deeply nested; &net.OpError{&os.SyscallError{syscall.Errno}}
	switch err := err.(type) {
	case syscall.Errno:
		return copyErrnoSeverity(err) // platform-specific

	case *os.SyscallError:
		return copyErrSeverity(err.Err)

	case *net.OpError:
		return copyErrSeverity(err.Err)

	default:
		return slog.WarnSeverity
	}
}

// HttpHandler transforms the given Service into a standard library HTTP handler. It is one of the main "bridges"
// between Typhon and net/http.
func HttpHandler(svc Service) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, httpReq *http.Request) {
		if httpReq.Body != nil {
			defer httpReq.Body.Close()
		}

		req := Request{
			Context: httpReq.Context(),
			Request: *httpReq}
		if h, ok := rw.(http.Hijacker); ok {
			req.hijacker = h
		}
		rsp := svc(req)

		// If the connection was hijacked, we should not attempt to write anything out
		if rsp.hijacked {
			return
		}

		rwHeader := rw.Header()
		for k, v := range rsp.Header {
			rwHeader[k] = v
		}
		rw.WriteHeader(rsp.StatusCode)
		if rsp.Body != nil {
			defer rsp.Body.Close()
			buf := *httpChunkBufPool.Get().(*[]byte)
			defer httpChunkBufPool.Put(&buf)
			if isStreamingRsp(rsp) {
				// Streaming responses use copyChunked(), which takes care of flushing transparently
				if _, err := copyChunked(rw, rsp.Body, buf); err != nil {
					slog.Log(slog.Eventf(copyErrSeverity(err), req, "Couldn't send streaming response body", err))

					// Prevent the client from accidentally consuming a truncated stream by aborting the response.
					// The official way of interrupting an HTTP reply mid-stream is panic(http.ErrAbortHandler), which
					// works for both HTTP/1.1 and HTTP.2. https://github.com/golang/go/issues/17790
					panic(http.ErrAbortHandler)
				}
			} else {
				if _, err := io.CopyBuffer(rw, rsp.Body, buf); err != nil {
					slog.Log(slog.Eventf(copyErrSeverity(err), req, "Couldn't send response body", err))
				}
			}
		}
	})
}
