package typhon

import (
	"crypto/tls"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type http2H2cFlavour struct {
	T      *testing.T
	client Service
}

func (f http2H2cFlavour) Serve(svc Service) *Server {
	svc = svc.Filter(H2cFilter)
	s, err := Listen(svc, "localhost:0")
	require.NoError(f.T, err)
	return s
}

func (f http2H2cFlavour) URL(s *Server) string {
	return fmt.Sprintf("http://%s", s.Listener().Addr())
}

func (f http2H2cFlavour) Proto() string {
	return "HTTP/2.0"
}

type http2H2Flavour struct {
	T      *testing.T
	client Service
	cert   tls.Certificate
}

func (f http2H2Flavour) Serve(svc Service) *Server {
	l, err := tls.Listen("tcp", "localhost:0", &tls.Config{
		Certificates: []tls.Certificate{f.cert},
		ClientAuth:   tls.NoClientCert,
		NextProtos:   []string{"h2"}})
	require.NoError(f.T, err)
	s, err := Serve(svc, l)
	require.NoError(f.T, err)
	return s
}

func (f http2H2Flavour) URL(s *Server) string {
	return fmt.Sprintf("https://%s", s.Listener().Addr())
}

func (f http2H2Flavour) Proto() string {
	return "HTTP/2.0"
}
