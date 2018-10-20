package typhon

import (
	"crypto/tls"
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
)

func TestMain(m *testing.M) {
	Client = Service(BareClient).Filter(ErrorFilter)
	os.Exit(m.Run())
}

type testUtils struct {
	T       *testing.T
	Client  Service
	filters []Filter
}

func (u testUtils) serve(svc Service) Server {
	for _, f := range u.filters {
		svc = svc.Filter(f)
	}
	s, err := Listen(svc, "localhost:0")
	require.NoError(u.T, err)
	return s
}

// testH1H2 runs the passed test function with both HTTP/1.1 and HTTP/2 implementations
func testH1H2(t *testing.T, f func(*testing.T, testUtils)) {
	t.Run("http1.1", func(t *testing.T) {
		f(t, testUtils{
			T:      t,
			Client: Service(BareClient).Filter(ErrorFilter)})
	})
	t.Run("http2.0-h2c", func(t *testing.T) {
		f(t, testUtils{
			T: t,
			Client: HttpService(&http2.Transport{
				AllowHTTP: true,
				DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
					return net.Dial(network, addr)
				}}),
			filters: []Filter{
				H2cFilter}})
	})
}
