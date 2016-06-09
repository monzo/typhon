package typhon

import (
	"net/http"
	"strings"

	"github.com/mondough/terrors"
)

var (
	mapTerr2Status = map[string]int{
		terrors.ErrBadRequest:      http.StatusBadRequest,
		terrors.ErrUnauthorized:    http.StatusUnauthorized,
		terrors.ErrForbidden:       http.StatusForbidden,
		terrors.ErrNotFound:        http.StatusNotFound,
		terrors.ErrBadResponse:     http.StatusNotAcceptable,
		terrors.ErrTimeout:         http.StatusGatewayTimeout,
		terrors.ErrInternalService: http.StatusInternalServerError}
	mapStatus2Terr map[int]string
)

func init() {
	mapStatus2Terr = make(map[int]string, len(mapTerr2Status))
	for k, v := range mapTerr2Status {
		mapStatus2Terr[v] = k
	}
}

// ErrorStatusCode returns an HTTP status code for the error
func ErrorStatusCode(err error) int {
	code := terrors.Wrap(err, nil).(*terrors.Error).Code
	if c, ok := mapTerr2Status[strings.SplitN(code, ".", 2)[0]]; ok {
		return c
	}
	return http.StatusInternalServerError
}

// terr2StatusCode converts HTTP status codes to a roughly equivalent terrors' code
func status2TerrCode(code int) string {
	if c, ok := mapStatus2Terr[code]; ok {
		return c
	}
	return terrors.ErrInternalService
}
