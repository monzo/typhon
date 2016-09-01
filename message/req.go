package message

import (
	"fmt"
)

type request struct {
	message
}

func (r *request) Copy() Request {
	r.RLock()
	defer r.RUnlock()
	return &request{
		message: message{
			id:       r.id,
			payload:  r.payload,
			body:     r.body,
			service:  r.service,
			endpoint: r.endpoint,
			headers:  r.headersCopy()}}
}

func (r *request) String() string {
	return fmt.Sprintf("Request(%s)", r.Id())
}

func NewRequest() Request {
	return &request{}
}
