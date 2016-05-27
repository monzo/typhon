package httpcompat

import (
	"net/http"

	"github.com/mondough/terrors"
)

var (
	mapHttpToTerror map[int]string
	mapTerrorToHttp map[string]int
)

func init() {
	mapHttpToTerror = map[int]string{
		http.StatusBadRequest:     terrors.ErrBadRequest,
		http.StatusUnauthorized:   terrors.ErrUnauthorized,
		http.StatusForbidden:      terrors.ErrForbidden,
		http.StatusNotFound:       terrors.ErrNotFound,
		http.StatusNotAcceptable:  terrors.ErrBadResponse,
		http.StatusGatewayTimeout: terrors.ErrTimeout}
	mapTerrorToHttp = make(map[string]int, len(mapHttpToTerror))
	for k, v := range mapHttpToTerror {
		mapTerrorToHttp[v] = k
	}
}

func httpToTerror(status int) string {
	if v, ok := mapHttpToTerror[status]; ok {
		return v
	}
	return terrors.ErrInternalService
}

func terorrToHttp(code string) int {
	if v, ok := mapTerrorToHttp[code]; ok {
		return v
	}
	return http.StatusInternalServerError
}
