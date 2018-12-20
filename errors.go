package typhon

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/monzo/slog"
	"github.com/monzo/terrors"
	"github.com/monzo/terrors/proto"
)

var (
	mapTerr2Status = map[string]int{
		terrors.ErrBadRequest:         http.StatusBadRequest,          // 400
		terrors.ErrBadResponse:        http.StatusNotAcceptable,       // 406
		terrors.ErrForbidden:          http.StatusForbidden,           // 403
		terrors.ErrInternalService:    http.StatusInternalServerError, // 500
		terrors.ErrNotFound:           http.StatusNotFound,            // 404
		terrors.ErrPreconditionFailed: http.StatusPreconditionFailed,  // 412
		terrors.ErrTimeout:            http.StatusGatewayTimeout,      // 504
		terrors.ErrUnauthorized:       http.StatusUnauthorized,        // 401
	}
	mapStatus2Terr map[int]string
)

func init() {
	mapStatus2Terr = make(map[int]string, len(mapTerr2Status))
	for k, v := range mapTerr2Status {
		mapStatus2Terr[v] = k
	}
}

// ErrorStatusCode returns a HTTP status code for the given error.
//
// If the error is not a terror, this will always be 500 (Internal Server Error).
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

// ErrorFilter serialises and deserialises response errors. Without this filter, errors may not be passed across
// the network properly so it is recommended to use this in most/all cases.
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
		rsp.Response = newHTTPResponse(req)
	}
	if rsp.Request == nil {
		rsp.Request = &req
	}

	if rsp.Error != nil {
		if rsp.StatusCode == http.StatusOK {
			// We got an error, but there is no error in the underlying response; marshal
			if rsp.Body != nil {
				rsp.Body.Close()
			}
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
				slog.Warn(rsp.Request, "Failed to unmarshal terror: %v", err)
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
