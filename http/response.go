package httpsvc

import (
	"net/http"
)

type Response struct {
	http.Response
	Error error
}
