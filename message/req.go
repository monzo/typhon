package message

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

func NewRequest() Request {
	return &request{}
}
