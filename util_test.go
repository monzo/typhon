package typhon

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	Client = Service(BareClient).Filter(ErrorFilter)
	os.Exit(m.Run())
}

func serve(t *testing.T, svc Service) Server {
	s, err := Listen(svc, "localhost:0")
	require.NoError(t, err)
	return s
}
