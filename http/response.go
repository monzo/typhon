package httpsvc

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
)

type Response struct {
	http.Response
	Error error
}

func NewResponse(status int, body []byte) Response {
	b := io.ReadCloser(nil)
	if body != nil {
		b = ioutil.NopCloser(bytes.NewReader(body))
	}

	return Response{
		Response: http.Response{
			Status:     http.StatusText(status),
			StatusCode: status,
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Body:       b}}
}
