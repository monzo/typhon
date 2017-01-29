package typhon

import (
	"testing"

	"github.com/monzo/terrors"
	"github.com/stretchr/testify/assert"
)

func TestRouter(t *testing.T) {
	t.Parallel()

	router := NewRouter()
	router.GET("/foo", func(req Request) Response {
		rsp := NewResponse(req)
		rsp.Write([]byte("abcdef"))
		return rsp
	})

	// Matching path
	req := NewRequest(nil, "GET", "/foo", nil)
	rsp := router.Serve()(req)
	assert.NoError(t, rsp.Error)
	b, _ := rsp.BodyBytes(true)
	assert.Equal(t, "abcdef", string(b))

	// Wrong method should result in not found
	// @TODO: This should really be HTTP Method Not Found
	req = NewRequest(nil, "POST", "/foo", nil)
	rsp = router.Serve()(req)
	assert.Error(t, rsp.Error)
	err := terrors.Wrap(rsp.Error, nil).(*terrors.Error)
	assert.True(t, err.Matches(terrors.ErrNotFound))

	// Wrong path should result in not found
	req = NewRequest(nil, "GET", "/", nil)
	rsp = router.Serve()(req)
	assert.Error(t, rsp.Error)
	err = terrors.Wrap(rsp.Error, nil).(*terrors.Error)
	assert.True(t, err.Matches(terrors.ErrNotFound))
}

func TestRouter_CatchallPath(t *testing.T) {
	t.Parallel()

	// Registering a global handler should apply to all paths
	router := NewRouter()
	router.GET("/*residual", func(req Request) Response {
		rsp := NewResponse(req)
		rsp.Write([]byte("catchall"))
		return rsp
	})
	req := NewRequest(nil, "GET", "/bar/baz/doodad/123/abc", nil)
	rsp := router.Serve()(req)
	assert.NoError(t, rsp.Error)
	b, _ := rsp.BodyBytes(true)
	assert.Equal(t, "catchall", string(b))
	// …but not on another method
	req = NewRequest(nil, "POST", "/foo", nil)
	rsp = router.Serve()(req)
	assert.Error(t, rsp.Error)
	err := terrors.Wrap(rsp.Error, nil).(*terrors.Error)
	assert.True(t, err.Matches(terrors.ErrNotFound))
}

// Validate that partial matches are resolved correctly
func TestRouterPartials(t *testing.T) {
	t.Parallel()

	router := NewRouter()
	router.GET("/foo/", func(req Request) Response {
		return req.Response("/foo/")
	})
	router.GET("/:param/bar", func(req Request) Response {
		return req.Response("/:param/bar")
	})

	req := NewRequest(nil, "GET", "/foo/", nil)
	rsp := router.Serve()(req)
	s := ""
	rsp.Decode(&s)
	assert.Equal(t, "/foo/", s)

	req = NewRequest(nil, "GET", "/abc/bar", nil)
	rsp = router.Serve()(req)
	rsp.Decode(&s)
	assert.Equal(t, "/:param/bar", s)
}
