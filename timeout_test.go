package typhon

import (
	"context"
	"testing"
	"time"

	"github.com/monzo/terrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeoutFilter(t *testing.T) {
	t.Parallel()
	// A Service which does not time out should be unmolested
	svc := Service(func(req Request) Response {
		return Response{}
	})
	svc = svc.Filter(TimeoutFilter(time.Second))
	rsp := svc(NewRequest(nil, "GET", "/", nil))
	assert.NoError(t, rsp.Error)

	// One which does should time out
	svc = Service(func(req Request) Response {
		time.Sleep(50 * time.Millisecond)
		return Response{}
	})
	svc = svc.Filter(TimeoutFilter(10 * time.Millisecond))
	rsp = svc(NewRequest(nil, "GET", "/", nil))
	require.Error(t, rsp.Error)
	assert.True(t, terrors.Wrap(rsp.Error, nil).(*terrors.Error).Matches(terrors.ErrTimeout))
}

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
