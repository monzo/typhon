package httpsvc

import (
	"encoding/json"
	"errors"

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
		rsp.StatusCode = terr2StatusCode(terr.Code)
		rsp.Header.Set("Terror", "1")
	} else if rsp.StatusCode >= 400 && rsp.StatusCode <= 599 {
		// Deserialise the response error
		b, err := rsp.BodyBytes(false)
		if err == nil {
			if rsp.Header.Get("Terror") == "1" {
				tp := &terrorsproto.Error{}
				if err := json.Unmarshal(b, tp); err == nil {
					rsp.Error = terrors.Unmarshal(tp)
				}
			}
			if rsp.Error == nil {
				rsp.Error = errors.New(string(b))
			}
		}
	}

	return rsp
}
