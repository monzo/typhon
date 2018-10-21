package typhon

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type http1Flavour struct {
	T *testing.T
}

func (f http1Flavour) Serve(svc Service) Server {
	s, err := Listen(svc, "localhost:0")
	require.NoError(f.T, err)
	return s
}
