package message

import (
	"fmt"
)

type response struct {
	message
}

func (r *response) Copy() Response {
	r.RLock()
	defer r.RUnlock()
	return &response{
		message: message{
			id:       r.id,
			payload:  r.payload,
			body:     r.body,
			service:  r.service,
			endpoint: r.endpoint,
			headers:  r.headersCopy(),
		}}
}

func (r *response) String() string {
	return fmt.Sprintf("Response(%s)", r.Id())
}

func NewResponse() Response {
	return &response{}
}
