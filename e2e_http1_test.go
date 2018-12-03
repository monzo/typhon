package typhon

import (
	"crypto/tls"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type http1Flavour struct {
	T *testing.T
}

func (f http1Flavour) Serve(svc Service) *Server {
	s, err := Listen(svc, "localhost:0")
	require.NoError(f.T, err)
	return s
}

func (f http1Flavour) URL(s *Server) string {
	return fmt.Sprintf("http://%s", s.Listener().Addr())
}

func (f http1Flavour) Proto() string {
	return "HTTP/1.1"
}

type http1TLSFlavour struct {
	T    *testing.T
	cert tls.Certificate
}

func (f http1TLSFlavour) Serve(svc Service) *Server {
	l, err := tls.Listen("tcp", "localhost:0", &tls.Config{
		Certificates: []tls.Certificate{f.cert},
		ClientAuth:   tls.NoClientCert})
	require.NoError(f.T, err)
	s, err := Serve(svc, l)
	require.NoError(f.T, err)
	return s
}

func (f http1TLSFlavour) URL(s *Server) string {
	return fmt.Sprintf("https://%s", s.Listener().Addr())
}

func (f http1TLSFlavour) Proto() string {
	return "HTTP/1.1"
}
