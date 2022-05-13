package typhon

import (
	"context"
	"crypto/tls"
	"fmt"
	"testing"

	"github.com/monzo/terrors"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

type http2H2cFlavour struct {
	T      *testing.T
	client Service
}

func (f http2H2cFlavour) Serve(svc Service, opts ...ServerOption) *Server {
	svc = svc.Filter(H2cFilter)
	s, err := Listen(svc, "localhost:0", opts...)
	require.NoError(f.T, err)
	return s
}

func (f http2H2cFlavour) URL(s *Server) string {
	return fmt.Sprintf("http://%s", s.Listener().Addr())
}

func (f http2H2cFlavour) Proto() string {
	return "HTTP/2.0"
}

func (f http2H2cFlavour) Context() (context.Context, func()) {
	return context.WithCancel(context.Background())
}

func (f http2H2cFlavour) AssertConnectionResetError(t *testing.T, terr *terrors.Error) {
	assert.Equal(t, terrors.ErrInternalService, terr.Code)
	assert.Contains(t, terr.Message, "INTERNAL_ERROR")
}

type http2H2cPriorKnowledgeFlavour struct {
	T      *testing.T
	client Service
}

func (f http2H2cPriorKnowledgeFlavour) Serve(svc Service, opts ...ServerOption) *Server {
	svc = svc.Filter(H2cFilter)
	s, err := Listen(svc, "localhost:0", opts...)
	require.NoError(f.T, err)
	return s
}

func (f http2H2cPriorKnowledgeFlavour) URL(s *Server) string {
	return fmt.Sprintf("http://%s", s.Listener().Addr())
}

func (f http2H2cPriorKnowledgeFlavour) Proto() string {
	return "HTTP/2.0"
}

func (f http2H2cPriorKnowledgeFlavour) Context() (context.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = WithH2C(ctx)
	return ctx, cancel
}

func (f http2H2cPriorKnowledgeFlavour) AssertConnectionResetError(t *testing.T, terr *terrors.Error) {
	assert.Equal(t, terrors.ErrInternalService, terr.Code)
	assert.Equal(t, "EOF", terr.Message)
}

type http2H2Flavour struct {
	T      *testing.T
	client Service
	cert   tls.Certificate
}

func (f http2H2Flavour) Serve(svc Service, opts ...ServerOption) *Server {
	l, err := tls.Listen("tcp", "localhost:0", &tls.Config{
		Certificates: []tls.Certificate{f.cert},
		ClientAuth:   tls.NoClientCert,
		NextProtos:   []string{"h2"}})
	require.NoError(f.T, err)
	s, err := Serve(svc, l, opts...)
	require.NoError(f.T, err)
	return s
}

func (f http2H2Flavour) URL(s *Server) string {
	return fmt.Sprintf("https://%s", s.Listener().Addr())
}

func (f http2H2Flavour) Proto() string {
	return "HTTP/2.0"
}

func (f http2H2Flavour) Context() (context.Context, func()) {
	return context.WithCancel(context.Background())
}

func (f http2H2Flavour) AssertConnectionResetError(t *testing.T, terr *terrors.Error) {
	assert.Equal(t, terrors.ErrInternalService, terr.Code)
	assert.Contains(t, terr.Message, "INTERNAL_ERROR")
}
