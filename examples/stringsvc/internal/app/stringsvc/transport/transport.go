package transport

import (
	"github.com/monzo/terrors"
	"github.com/monzo/typhon"
	"github.com/monzo/typhon/examples/stringsvc/internal/app/stringsvc/service"
	"github.com/monzo/typhon/examples/stringsvc/pkg/stringsvc"
	"github.com/monzo/typhon/examples/stringsvc/pkg/stringsvc/transport"
)

func handleUppercase(svc stringsvc.Service) typhon.Service {
	return func(req typhon.Request) typhon.Response {
		request := transport.UppercaseRequest{}
		if err := req.Decode(&request); err != nil {
			resp := req.Response(nil)
			resp.Error = err
			return resp
		}

		result, err := svc.Uppercase(request.S)
		if err != nil {
			resp := req.Response(nil)
			switch err {
			case service.ErrEmpty:
				resp.Error = terrors.BadRequest("empty_string", err.Error(), nil)
			default:
				resp.Error = terrors.InternalService("", err.Error(), nil)
			}
			return resp
		}

		return req.Response(transport.UppercaseResponse{Value: result})
	}
}

func handleCount(svc stringsvc.Service) typhon.Service {
	return func(req typhon.Request) typhon.Response {
		request := transport.CountRequest{}
		if err := req.Decode(&request); err != nil {
			resp := req.Response(nil)
			resp.Error = err
			return resp
		}

		result, _ := svc.Count(request.S)

		return req.Response(transport.CountResponse{Value: result})
	}
}

func NewHTTPTransport(service stringsvc.Service) typhon.Service {
	router := typhon.Router{}

	router.POST("/uppercase", handleUppercase(service))
	router.POST("/count", handleCount(service))

	return router.Serve()
}
