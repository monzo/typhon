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
		message: *(r.message.copy()),
	}
}

func (r *request) String() string {
	return fmt.Sprintf("Request(%s)", r.Id())
}

func NewRequest() Request {
	return &request{}
}
