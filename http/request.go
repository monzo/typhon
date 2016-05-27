package httpsvc

import (
	"net/http"

	"golang.org/x/net/context"
)

// @TODO: Propagate context cancellation
type Request struct {
	http.Request
	context.Context
}

func NewRequest(ctx context.Context, method, url string) Request {
	if ctx == nil {
		ctx = context.Background()
	}
	req, _ := http.NewRequest(method, url, nil) // @TODO: Don't swallow this error
	return Request{
		Request: *req,
		Context: ctx}
}
