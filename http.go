package typhon

import (
	"context"
	"io"
	"net/http"

	"github.com/monzo/slog"
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

func HttpHandler(svc Service) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, httpReq *http.Request) {
		ctx, cancel := context.WithCancel(httpReq.Context())
		defer cancel() // if already cancelled on escape, this is a no-op

		// If the ResponseWriter is a CloseNotifier, propagate the cancellation downward via the context
		if cn, ok := rw.(http.CloseNotifier); ok {
			closed := cn.CloseNotify()
			go func() {
				select {
				case <-ctx.Done():
				case <-closed:
					cancel()
				}
			}()
		}

		if httpReq.Body != nil {
			defer httpReq.Body.Close()
		}

		req := Request{
			Context: ctx,
			Request: *httpReq}
		rsp := svc(req)

		// Write the response out
		for k, v := range rsp.Header {
			if k == "Content-Length" {
				continue
			}
			rw.Header()[k] = v
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
