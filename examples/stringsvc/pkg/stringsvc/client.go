package stringsvc

import (
	"context"
	"github.com/monzo/typhon"
	"github.com/monzo/typhon/examples/stringsvc/pkg/stringsvc/transport"
)

type client struct {
	baseUrl      string
	typhonClient typhon.Service
}

func NewClient(baseUrl string, typhonClient typhon.Service) Service {
	return &client{
		baseUrl:      baseUrl,
		typhonClient: typhonClient,
	}
}

func (c client) Uppercase(s string) (string, error) {
	req := typhon.NewRequest(context.Background(), "POST", c.baseUrl+"/uppercase", transport.UppercaseRequest{S: s})
	response := req.SendVia(c.typhonClient).Response()

	resp := transport.UppercaseResponse{}
	if err := response.Decode(&resp); err != nil {
		return "", err
	}

	return resp.Value, nil
}

func (c client) Count(s string) (int, error) {
	req := typhon.NewRequest(context.Background(), "POST", c.baseUrl+"/count", transport.CountRequest{S: s})
	response := req.SendVia(c.typhonClient).Response()

	resp := transport.CountResponse{}
	if err := response.Decode(&resp); err != nil {
		return 0, err
	}

	return resp.Value, nil
}
