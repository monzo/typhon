package typhon

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	legacyproto "github.com/golang/protobuf/proto"
	"github.com/monzo/slog"
	"github.com/monzo/terrors"
	terrorsproto "github.com/monzo/terrors/proto"
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
		terrors.ErrRateLimited:        http.StatusTooManyRequests,     // 429
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
// It tries to do everything it can to give you all the information that it can about why your request might have failed.
// Because of this, it has some weird behavior.
func ErrorFilter(req Request, svc Service) Response {
	var rsp Response

	// If the request contains an error, short-circuit and don't apply the given Service to the request.
	// req.err being non-nil normally means we got an error trying to constructing the underlying http.Request and so there
	// is no request to pass through to svc.
	if req.err != nil {
		rsp = NewResponse(req)
		rsp.Error = req.err
	} else {
		// rsp.Error could be non-nil. This normally represents an error during the round trip (e.g. connection closed).
		// It isn't expected to repsent an error received from the remote.
		rsp = svc(req)

		// If we never got a response then construct a default response. This is only expected to be needed if rsp.Error is non-nil.
		if rsp.Response == nil {
			// Status defaults to StatusOK here. It will be updated to the correct status later in the function.
			rsp.Response = newHTTPResponse(req, http.StatusOK)
		}
	}

	// ErrorFilter tries to make sure there is as much information as possible to debug the fault.
	// If for some reason the Request is not currently set (e.g. if it was lost by another Filter) then re-set it to req.
	if rsp.Request == nil {
		rsp.Request = &req
	}

	// If there is an error we want to turn it into a Terror on the response.
	if rsp.Error != nil {
		// We should be here if we hit one of the first two cases in this function.
		// We could also be here if something weird happened e.g. an error was set and a 200 response was returned by the server.
		if rsp.StatusCode == http.StatusOK {
			// We got an error, but there is no error in the underlying response; marshal
			if rsp.Body != nil {
				rsp.Body.Close()
			}
			rsp.Body = &bufCloser{}
			terr := terrors.Wrap(rsp.Error, nil).(*terrors.Error)
			rsp.Encode(terrors.Marshal(terr))
			// We now set the status to the ACTUAL status code based on the Terror.
			rsp.StatusCode = ErrorStatusCode(terr)
			rsp.Header.Set("Terror", "1")
		}
	} else if rsp.StatusCode >= 400 && rsp.StatusCode <= 599 {
		// There is an error in the underlying response; unmarshal
		b, _ := rsp.BodyBytes(false)
		switch rsp.Header.Get("Terror") {
		case "1":
			var err error
			tp := &terrorsproto.Error{}

			switch rsp.Header.Get("Content-Type") {
			case "application/octet-stream", "application/x-protobuf", "application/protobuf":
				err = legacyproto.Unmarshal(b, tp)
			default:
				err = json.Unmarshal(b, tp)
			}

			if err != nil {
				slog.Warn(rsp.Request, "Failed to unmarshal terror: %v", err)
				rsp.Error = errors.New(string(b))
			} else {
				rsp.Error = terrors.Unmarshal(tp)
			}
		default:
			rsp.Error = errors.New(string(b))
		}
	}

	// If there was an error but the error string is empty re-write the error with the information we have.
	if rsp.Error != nil && rsp.Error.Error() == "" {
		if rsp.Response != nil {
			rsp.Error = fmt.Errorf("Response error (%d)", rsp.StatusCode)
		} else {
			// We don't expect to ever hit this case right now.
			rsp.Error = fmt.Errorf("Response error")
		}
	}

	return rsp
}
