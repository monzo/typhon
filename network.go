package typhon

import (
	"encoding/json"
	"errors"

	log "github.com/mondough/slog"
	"github.com/mondough/terrors"
	"github.com/mondough/terrors/proto"
)

// networkFilter prepares the request and response for network transport
func networkFilter(req Request, svc Service) Response {
	rsp := svc(req)
	rsp.ctx = req.Context

	if rsp.Response == nil {
		// Errors often won't have a HTTP response
		rsp.Response = newHttpResponse(req)
	}

	if rsp.Error != nil {
		// Serialise the error into the response
		terr := terrors.Wrap(rsp.Error, nil).(*terrors.Error)
		rsp.Encode(terrors.Marshal(terr))
		rsp.StatusCode = ErrorStatusCode(terr)
		rsp.Header.Set("Terror", "1")
	} else if rsp.StatusCode >= 400 && rsp.StatusCode <= 599 && rsp.Header.Get("Terror") == "1" {
		b, _ := rsp.BodyBytes(false)
		tp := &terrorsproto.Error{}
		if err := json.Unmarshal(b, tp); err != nil {
			log.Warn(req, "Failed to unmarshal terror: %v", err)
			rsp.Error = errors.New(string(b))
		} else {
			rsp.Error = terrors.Unmarshal(tp)
		}
	} else if rsp.StatusCode >= 500 && rsp.StatusCode <= 599 {
		b, _ := rsp.BodyBytes(false)
		rsp.Error = errors.New(string(b))
	}

	return rsp
}
