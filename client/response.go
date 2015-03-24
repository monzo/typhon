package client

// Response represents a response from a service following an RPC call
type Response interface {
	// ContentType of the payload
	ContentType() string
	// Service the response is from
	Service() string
	// Endpoint the response is from
	Endpoint() string
	// Payload stores our raw response
	Payload() []byte
	// Is the response an Error
	IsError() bool
}

// response is our concrete implementation
//
// @todo push creation of this down into the transport layer
// and just use the interface within the client package
// so that the client has no knowledge of the internals of the transport
type response struct {
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
func (r *response) ContentType() string {
	return r.contentType
}

// Service the response came from
func (r *response) Service() string {
	return r.service
}

// Endpoint the response came from
func (r *response) Endpoint() string {
	return r.endpoint
}

// Payload of the response
func (r *response) Payload() []byte {
	return r.payload
}

// IsError determines if the response is an error
func (r *response) IsError() bool {
	return r.contentEncoding == "error"
}
