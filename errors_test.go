package typhon

import (
	"context"
	"net/http"
	"testing"

	"github.com/monzo/terrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ErrorFilter must keep the response error whatever encoding the caller asked for.
func TestErrorFilterKeepsErrorForAllAcceptTypes(t *testing.T) {
	t.Parallel()

	svc := Service(func(req Request) Response {
		return Response{Error: terrors.NotFound("thing", "thing not found", nil)}
	}).Filter(ErrorFilter)

	for _, accept := range []string{"application/json", "application/protobuf"} {
		accept := accept
		t.Run(accept, func(t *testing.T) {
			req := NewRequest(context.Background(), "GET", "/", nil)
			req.Header.Set("Accept", accept)

			rsp := svc(req)
			assert.Equal(t, http.StatusNotFound, rsp.StatusCode)
			require.Error(t, rsp.Error)
			assert.True(t, terrors.Is(rsp.Error, "not_found.thing"))
		})
	}
}