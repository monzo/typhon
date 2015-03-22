package client

type Request interface {
	// ContentType of the payload
	ContentType() string
	// Service to be delivered to
	Service() string
	// Endpoint to be delivered to at the service
	Endpoint() string
	// Payload stores our raw payload to send
	Payload() []byte
}

type request struct {
	// contentType of the payload
	contentType string
	// service to be delivered to
	service string
	// endpoint to be delivered to at the service
	endpoint string
	// payload stores our raw payload to send
	payload []byte
}

// ContentType of the request
func (r *request) ContentType() string {
	return r.contentType
}

// Service to be delivered to
func (r *request) Service() string {
	return r.service
}

// Endpoint to be delivered to at the service
func (r *request) Endpoint() string {
	return r.endpoint
}

// Payload of the request
func (r *request) Payload() []byte {
	return r.payload
}

// NewProtoRequest creates a new request with protobuf encoding
func NewProtoRequest(service, endpoint string, payload []byte) Request {
	return &request{
		contentType: "application/x-protobuf",
		service:     service,
		endpoint:    endpoint,
		payload:     payload,
	}
}

// NewJsonRequest creates a new request with json encoding
func NewJsonRequest(service, endpoint string, payload []byte) Request {
	return &request{
		contentType: "application/json",
		service:     service,
		endpoint:    endpoint,
		payload:     payload,
	}
}
