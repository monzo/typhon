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
		message: *(r.message.copy()),
	}
}

func (r *response) String() string {
	return fmt.Sprintf("Response(%s)", r.Id())
}

func NewResponse() Response {
	return &response{}
}
