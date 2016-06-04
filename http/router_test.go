package httpsvc

import (
	"testing"

	"github.com/mondough/terrors"
	"github.com/stretchr/testify/assert"
)

func TestRouter(t *testing.T) {
	t.Parallel()

	router := NewRouter()
	router.GET("/foo", func(req Request) Response {
		rsp := NewResponse(req)
		rsp.SetBody([]byte("abcdef"))
		return rsp
	})

	// Matching path
	req := NewRequest(nil, "GET", "/foo")
	rsp := router.Serve()(req)
	assert.NoError(t, rsp.Error)
	assert.Equal(t, "abcdef", string(rsp.BodyBytes()))

	// Wrong method should result in not found
	// @TODO: This should really be HTTP Method Not Found
	req = NewRequest(nil, "POST", "/foo")
	rsp = router.Serve()(req)
	assert.Error(t, rsp.Error)
	err := terrors.Wrap(rsp.Error, nil).(*terrors.Error)
	assert.True(t, err.Matches(terrors.ErrNotFound))

	// Wrong path should result in not found
	req = NewRequest(nil, "GET", "/")
	rsp = router.Serve()(req)
	assert.Error(t, rsp.Error)
	err = terrors.Wrap(rsp.Error, nil).(*terrors.Error)
	assert.True(t, err.Matches(terrors.ErrNotFound))
}

func TestRouter_CatchallPath(t *testing.T) {
	// Registering a global handler should apply to all paths
	router := NewRouter()
	router.GET("/*residual", func(req Request) Response {
		rsp := NewResponse(req)
		rsp.SetBody([]byte("catchall"))
		return rsp
	})
	req := NewRequest(nil, "GET", "/bar/baz/doodad/123/abc")
	rsp := router.Serve()(req)
	assert.NoError(t, rsp.Error)
	assert.Equal(t, "catchall", string(rsp.BodyBytes()))
	// …but not on another method
	req = NewRequest(nil, "POST", "/foo")
	rsp = router.Serve()(req)
	assert.Error(t, rsp.Error)
	err := terrors.Wrap(rsp.Error, nil).(*terrors.Error)
	assert.True(t, err.Matches(terrors.ErrNotFound))
}
