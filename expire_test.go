package typhon

import (
	"context"
	"net/http"
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

	// An unexpired request should be allowed through
	req := NewRequest(context.Background(), "GET", "/", nil)
	rsp := svc(req)
	assert.NoError(t, rsp.Error)
	assert.Equal(t, http.StatusOK, rsp.StatusCode)
	b, err := rsp.BodyBytes(true)
	require.NoError(t, err)
	assert.Equal(t, []byte(`"ok"`+"\n"), b)

	// An expired request should be rejected
	ctx, ccl := context.WithCancel(context.Background())
	ccl()
	req.Context = ctx
	rsp = svc(req)
	assert.Error(t, rsp.Error)
	terr := terrors.Wrap(rsp.Error, nil).(*terrors.Error)
	terrExpect := terrors.BadRequest("expired", "Request has expired", nil)
	assert.Equal(t, terrExpect.Message, terr.Message)
	assert.Equal(t, terrExpect.Code, terr.Code)
}
