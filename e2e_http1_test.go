package typhon

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/monzo/terrors"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/stretchr/testify/require"
)

type http1Flavour struct {
	T *testing.T
}

func (f http1Flavour) Serve(svc Service, opts ...ServerOption) *Server {
	s, err := Listen(svc, "localhost:0", opts...)
	require.NoError(f.T, err)
	return s
}

func (f http1Flavour) URL(s *Server) string {
	return fmt.Sprintf("http://%s", s.Listener().Addr())
}

func (f http1Flavour) Proto() string {
	return "HTTP/1.1"
}

func (f http1Flavour) Context() (context.Context, func()) {
	return context.WithCancel(context.Background())
}

func (f http1Flavour) AssertConnectionResetError(t *testing.T, terr *terrors.Error) {
	assert.Equal(t, terrors.ErrInternalService, terr.Code)
	assert.Equal(t, "EOF", terr.Message)
}

type http1TLSFlavour struct {
	T    *testing.T
	cert tls.Certificate
}

func (f http1TLSFlavour) Serve(svc Service, opts ...ServerOption) *Server {
	l, err := tls.Listen("tcp", "localhost:0", &tls.Config{
		Certificates: []tls.Certificate{f.cert},
		ClientAuth:   tls.NoClientCert})
	require.NoError(f.T, err)
	s, err := Serve(svc, l, opts...)
	require.NoError(f.T, err)
	return s
}

func (f http1TLSFlavour) URL(s *Server) string {
	return fmt.Sprintf("https://%s", s.Listener().Addr())
}

func (f http1TLSFlavour) Proto() string {
	return "HTTP/1.1"
}

func (f http1TLSFlavour) Context() (context.Context, func()) {
	return context.WithCancel(context.Background())
}

func (f http1TLSFlavour) AssertConnectionResetError(t *testing.T, terr *terrors.Error) {
	assert.Equal(t, terrors.ErrInternalService, terr.Code)
	assert.Equal(t, "local error: tls: bad record MAC", terr.Message)
}
