package message

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

func NewResponse() Response {
	return &response{}
}
