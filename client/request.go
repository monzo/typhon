package client

import (
	"time"

	log "github.com/cihub/seelog"
	"github.com/nu7hatch/gouuid"

	"github.com/b2aio/typhon/auth"
	"github.com/b2aio/typhon/errors"
)

// Request to be sent to another service
type Request interface {
	// Id of this request
	Id() string
	// ContentType of the payload
	ContentType() string
	// Service to be delivered to
	Service() string
	// Endpoint to be delivered to at the service
	Endpoint() string
	// Payload stores our raw payload to send
	Payload() []byte
	// Request timeout
	Timeout() time.Duration
	SetTimeout(time.Duration)
	// Session for this request containing authentication info
	Session() auth.Session
	SetSession(s auth.Session)
}

type request struct {
	// id of this request
	id string
	// contentType of the payload
	contentType string
	// service to be delivered to
	service string
	// endpoint to be delivered to at the service
	endpoint string
	// payload stores our raw payload to send
	payload []byte
	// request timeout
	timeout time.Duration
	// session used for authentication
	session auth.Session
}

// Id of the request
func (r *request) Id() string {
	return r.id
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

// Timeout of the request
func (r *request) Timeout() time.Duration {
	if r.timeout == 0 {
		return defaultTimeout
	}
	return r.timeout
}
func (r *request) SetTimeout(d time.Duration) {
	r.timeout = d
}

// Session stores authentication details
func (r *request) Session() auth.Session {
	return r.session
}

// SetSession information on this request
func (r *request) SetSession(s auth.Session) {
	r.session = s
}

// NewProtoRequest creates a new request with protobuf encoding
func NewProtoRequest(service, endpoint string, payload []byte) (Request, error) {
	requestId, err := uuid.NewV4()
	if err != nil {
		log.Errorf("[Client] Failed to create unique request id: %v", err)
		return nil, errors.Wrap(err) // @todo custom error code
	}

	return &request{
		id:          requestId.String(),
		contentType: "application/x-protobuf",
		service:     service,
		endpoint:    endpoint,
		payload:     payload,
	}, nil
}

// NewJsonRequest creates a new request with json encoding
func NewJsonRequest(service, endpoint string, payload []byte) (Request, error) {
	requestId, err := uuid.NewV4()
	if err != nil {
		log.Errorf("[Client] Failed to create unique request id: %v", err)
		return nil, errors.Wrap(err) // @todo custom error code
	}

	return &request{
		id:          requestId.String(),
		contentType: "application/json",
		service:     service,
		endpoint:    endpoint,
		payload:     payload,
	}, nil
}
