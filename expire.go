package typhon

import "github.com/monzo/terrors"

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
