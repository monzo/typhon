package typhon

import (
	"context"
	"testing"

	"github.com/monzo/terrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpirationFilter(t *testing.T) {
	t.Parallel()

	svc := Service(func(req Request) Response {
		return req.Response("ok")
	})
	svc = svc.Filter(ExpirationFilter)

	ctx, ccl := context.WithCancel(context.Background())
	ccl()
	req := NewRequest(ctx, "GET", "/", nil)
	rsp := svc(req)

	require.Error(t, rsp.Error)
	terr := terrors.Wrap(rsp.Error, nil).(*terrors.Error)
	terrExpect := terrors.BadRequest("expired", "Request has expired", nil)
	assert.Equal(t, terrExpect.Message, terr.Message)
	assert.Equal(t, terrExpect.Code, terr.Code)
}
