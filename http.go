package typhon

import (
	"context"
	"io"
	"net/http"

	log "github.com/monzo/slog"
)

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

		// Write the response out to the wire
		for k, v := range rsp.Header {
			if k == "Content-Length" {
				continue
			}
			rw.Header()[k] = v
		}
		rw.WriteHeader(rsp.StatusCode)
		if rsp.Body != nil {
			defer rsp.Body.Close()
			if _, err := io.Copy(rw, rsp.Body); err != nil {
				log.Error(req, "Error copying response body: %v", err)
			}
		}
	})
}

func HttpServer(svc Service) *http.Server {
	return &http.Server{
		Handler:        HttpHandler(svc),
		MaxHeaderBytes: http.DefaultMaxHeaderBytes}
}
