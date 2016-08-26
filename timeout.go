package typhon

import (
	"strconv"
	"time"

	"github.com/monzo/terrors"
	"golang.org/x/net/context"
)

func TimeoutFilter(defaultTimeout time.Duration) Filter {
	return func(req Request, svc Service) Response {
		timeout := defaultTimeout
		if t, err := strconv.Atoi(req.Header.Get("Timeout")); err == nil {
			timeout = time.Duration(t) * time.Millisecond
		}

		req.Context, _ = context.WithTimeout(req.Context, timeout)
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
