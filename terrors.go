package typhon

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	log "github.com/monzo/slog"
	"github.com/monzo/terrors"
	"github.com/monzo/terrors/proto"
)

var (
	mapTerr2Status = map[string]int{
		terrors.ErrBadRequest:         http.StatusBadRequest,
		terrors.ErrBadResponse:        http.StatusNotAcceptable,
		terrors.ErrForbidden:          http.StatusForbidden,
		terrors.ErrInternalService:    http.StatusInternalServerError,
		terrors.ErrNotFound:           http.StatusNotFound,
		terrors.ErrPreconditionFailed: http.StatusPreconditionFailed,
		terrors.ErrTimeout:            http.StatusGatewayTimeout,
		terrors.ErrUnauthorized:       http.StatusUnauthorized,
	}
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

// ErrorFilter serialises and de-serialises response errors
func ErrorFilter(req Request, svc Service) Response {
	// If the request contains an error, short-circuit and return that directly
	var rsp Response
	if req.err != nil {
		rsp = NewResponse(req)
		rsp.Error = req.err
	} else {
		rsp = svc(req)
	}

	if rsp.Response == nil {
		rsp.Response = newHttpResponse(req)
	}
	if rsp.ctx == nil {
		rsp.ctx = req
	}

	if rsp.Error != nil {
		if rsp.StatusCode == http.StatusOK {
			// We got an error, but there is no error in the underlying response; marshal
			rsp.Body = &bufCloser{}
			terr := terrors.Wrap(rsp.Error, nil).(*terrors.Error)
			rsp.Encode(terrors.Marshal(terr))
			rsp.StatusCode = ErrorStatusCode(terr)
			rsp.Header.Set("Terror", "1")
		}
	} else if rsp.StatusCode >= 400 && rsp.StatusCode <= 599 {
		// There is an error in the underlying response; unmarshal
		b, _ := rsp.BodyBytes(false)
		switch rsp.Header.Get("Terror") {
		case "1":
			tp := &terrorsproto.Error{}
			if err := json.Unmarshal(b, tp); err != nil {
				log.Warn(rsp.ctx, "Failed to unmarshal terror: %v", err)
				rsp.Error = errors.New(string(b))
			} else {
				rsp.Error = terrors.Unmarshal(tp)
			}
		default:
			rsp.Error = errors.New(string(b))
		}
	}

	if rsp.Error != nil && rsp.Error.Error() == "" {
		if rsp.Response != nil {
			rsp.Error = fmt.Errorf("Response error (%d)", rsp.StatusCode)
		} else {
			rsp.Error = fmt.Errorf("Response error")
		}
	}

	return rsp
}
