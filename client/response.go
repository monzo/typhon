package client

type Response struct {
	// contentType of the payload
	contentType string
	// contentEncoding is the encoding format of the returned response
	// eg. a response, or an encoded error
	contentEncoding string
	// service the response is from
	service string
	// endpoint the response is from
	endpoint string
	// payload stores our raw response
	payload []byte
}

// ContentType of the response
func (r *Response) ContentType() string {
	return r.contentType
}

// Service the response came from
func (r *Response) Service() string {
	return r.service
}

// Endpoint the response came from
func (r *Response) Endpoint() string {
	return r.endpoint
}

// Payload of the response
func (r *Response) Payload() []byte {
	return r.payload
}

func (r *Response) IsError() bool {
	return contentEncoding == "error"
}
