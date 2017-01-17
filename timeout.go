package typhon

import (
	"context"
	"time"

	"github.com/monzo/terrors"
)

// TimeoutFilter returns a Filter which will cancel a Request after the given timeout
func TimeoutFilter(timeout time.Duration) Filter {
	return func(req Request, svc Service) Response {
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

// ExpirationFilter provides admission control; it rejects requests which are cancelled
func ExpirationFilter(req Request, svc Service) Response {
	select {
	case <-req.Context.Done():
		return Response{
			Error: terrors.BadRequest("expired", "Request has expired", nil)}
	default:
		return svc(req)
	}
}
