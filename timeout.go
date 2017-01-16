package typhon

import (
	"context"
	"strconv"
	"time"

	"github.com/monzo/terrors"
)

// TimeoutFilter returns a Filter which will cancel a Request after the given timeout
func TimeoutFilter(defaultTimeout time.Duration) Filter {
	return func(req Request, svc Service) Response {
		timeout := defaultTimeout
		if t, err := strconv.Atoi(req.Header.Get("Timeout")); err == nil {
			timeout = time.Duration(t) * time.Millisecond
		}

		ctx, cancel := context.WithTimeout(req.Context, timeout)
		req.Context = ctx
		defer cancel()
		rspChan := make(chan Response, 1)
		go func() {
			rspChan <- svc(req)
		}()

		select {
		case rsp := <-rspChan:
			return rsp
		case <-req.Context.Done():
			return Response{
				Error: terrors.Timeout("", "Request timed out", nil)}
		}
	}
}
