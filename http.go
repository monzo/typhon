package typhon

import (
	"io"
	"net/http"

	"github.com/monzo/slog"
)

const (
	// chunkThreshold is a byte threshold above which request and response bodies that result from using buffered I/O
	// within Typhon will be transferred with chunked encoding on the wire.
	chunkThreshold = 5 * 1000000 // 5 megabytes
)

func isStreamingRsp(rsp Response) bool {
	// Most straightforward: service may have set rsp.Body to a streamer
	if s, ok := rsp.Body.(*streamer); ok && s != nil {
		return true
	}
	// In a proxy situation, the upstream would have set Transfer-Encoding
	for _, v := range rsp.Header["Transfer-Encoding"] {
		if v == "chunked" {
			return true
		}
	}
	// Annoyingly, this can be removed from headers by net/http and promoted to its own field
	for _, v := range rsp.TransferEncoding {
		if v == "chunked" {
			return true
		}
	}
	return false
}

// HttpHandler transforms the given Service into a http.Handler, suitable for use directly with net/http
func HttpHandler(svc Service) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, httpReq *http.Request) {
		if httpReq.Body != nil {
			defer httpReq.Body.Close()
		}

		req := Request{
			Context: httpReq.Context(),
			Request: *httpReq}
		rsp := svc(req)

		// Write the response out
		rwHeader := rw.Header()
		for k, v := range rsp.Header {
			if k == "Content-Length" {
				continue
			}
			rwHeader[k] = v
		}
		rw.WriteHeader(rsp.StatusCode)
		if rsp.Body != nil {
			defer rsp.Body.Close()
			if isStreamingRsp(rsp) {
				// Streaming responses use copyChunked(), which takes care of flushing transparently
				if _, err := copyChunked(rw, rsp.Body); err != nil {
					slog.Error(req, "Error copying streaming response body: %v", err)
				}
			} else {
				if _, err := io.Copy(rw, rsp.Body); err != nil {
					slog.Error(req, "Error copying response body: %v", err)
				}
			}
		}
	})
}

func HttpServer(svc Service) *http.Server {
	return &http.Server{
		Handler:        HttpHandler(svc),
		MaxHeaderBytes: http.DefaultMaxHeaderBytes}
}
