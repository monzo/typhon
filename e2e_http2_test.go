package typhon

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type http2H2cFlavour struct {
	T      *testing.T
	client Service
}

func (f http2H2cFlavour) Serve(svc Service) Server {
	svc = svc.Filter(H2cFilter)
	s, err := Listen(svc, "localhost:0")
	require.NoError(f.T, err)
	return s
}
