package httpsvc

import (
	"encoding/json"
	"io"
	"net/http"

	log "github.com/mondough/slog"
	"github.com/mondough/terrors"
)

type ResponseWriter interface {
	http.ResponseWriter
	// WriteJSON writes the given data as JSON to the Response. The passed value must (perhaps obviously) be
	// serialisable to JSON.
	WriteJSON(interface{})
	// WriteError writes the given error to the Response.
	WriteError(err error)
}

type responseWriterWrapper struct {
	r *Response
}

func (rw responseWriterWrapper) Header() http.Header {
	return rw.r.Header
}

func (rw responseWriterWrapper) Write(body []byte) (int, error) {
	switch rc := rw.r.Body.(type) {
	// In the "regular" case, the response body will be a bufCloser; we can write
	case io.Writer:
		return rc.Write(body)
	// If a caller manually sets Response.Body, then we may not be able to write to it. In that case, we need to be
	// cleverer.
	default:
		buf := &bufCloser{}
		if _, err := io.Copy(buf, rc); err != nil {
			// This can be quite bad; we have consumed (and possibly lost) some of the original body
			return 0, err
		}
		rw.r.Body = buf
		return buf.Write(body)
	}
}

func (rw responseWriterWrapper) WriteHeader(status int) {
	rw.r.StatusCode = status
}

func (rw responseWriterWrapper) WriteJSON(body interface{}) {
	rw.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(rw).Encode(body); err != nil {
		log.Warn(rw.r.Context, "Could not serialise JSON response: %v", err)
		terr := terrors.Wrap(err, nil).(*terrors.Error)
		terr.Code = terrors.ErrBadResponse
		rw.WriteError(terr)
	}
}

func (rw responseWriterWrapper) WriteError(err error) {
	rw.r.Error = err
}
