package httpsvc

import (
	"testing"
	"time"

	"github.com/mondough/terrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeout(t *testing.T) {
	t.Parallel()
	// A Service which does not time out should be unmolested
	svc := Service(func(req Request) Response {
		return Response{}
	})
	svc = svc.Filtered(TimeoutFilter(10 * time.Second))
	rsp := svc(NewRequest(nil, "GET", "/"))
	assert.NoError(t, rsp.Error)

	// One which does should timeout with the default timeout
	svc = Service(func(req Request) Response {
		time.Sleep(50 * time.Millisecond)
		return Response{}
	})
	svc = svc.Filtered(TimeoutFilter(10 * time.Millisecond))
	rsp = svc(NewRequest(nil, "GET", "/"))
	require.Error(t, rsp.Error)
	assert.True(t, terrors.Wrap(rsp.Error, nil).(*terrors.Error).Matches(terrors.ErrTimeout))

	// …or the one in the request if one was specified
	req := NewRequest(nil, "GET", "/")
	req.Header.Set("Timeout", "100") // 100 milliseconds
	rsp = svc(req)
	assert.NoError(t, rsp.Error)
	req.Header.Set("Timeout", "5")
	rsp = svc(NewRequest(nil, "GET", "/"))
	require.Error(t, rsp.Error)
	assert.True(t, terrors.Wrap(rsp.Error, nil).(*terrors.Error).Matches(terrors.ErrTimeout))
}
