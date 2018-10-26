package typhon

import (
	"context"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func BenchmarkRequestResponse(b *testing.B) {
	b.ReportAllocs()
	svc := Service(func(req Request) Response {
		rsp := req.Response(nil)
		rsp.Header.Set("a", "b")
		rsp.Header.Set("b", "b")
		rsp.Header.Set("c", "b")
		return rsp
	})
	addr := &net.UnixAddr{
		Net:  "unix",
		Name: "/tmp/typhon-test.sock"}
	l, _ := net.ListenUnix("unix", addr)
	defer l.Close()
	s, _ := Serve(svc, l)
	defer s.Stop(context.Background())

	conn, err := net.DialUnix("unix", nil, addr)
	require.NoError(b, err)
	defer conn.Close()
	sockTransport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return conn, nil
		}}

	ctx := context.Background()
	req := NewRequest(ctx, "GET", "http://localhost/foo", nil)
	sockSvc := HttpService(sockTransport)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req.SendVia(sockSvc).Response()
	}
}
