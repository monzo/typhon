package server

import (
	"github.com/streadway/amqp"
	"golang.org/x/net/context"

	log "github.com/cihub/seelog"
	"github.com/obeattie/typhon/auth"
)

// AMQPRequest satisfies our server Request interface, and
// represents requests delivered via an AMQP implementation
type AMQPRequest struct {
	context.Context

	// reference to the server handling this request
	s Server

	body     interface{}
	delivery *amqp.Delivery

	// session holds our authentication information
	session auth.Session

	id              string
	contentType     string
	contentEncoding string
	service         string
	endpoint        string
	accessToken     string
	traceID         string
	parentRequestID string
}

// NewAMQPRequest marshals a raw AMQP delivery to a Request interface
func NewAMQPRequest(s Server, delivery *amqp.Delivery) (*AMQPRequest, error) {
	// Handle basic headers
	contentType, _ := delivery.Headers["Content-Type"].(string)
	contentEncoding, _ := delivery.Headers["Content-Encoding"].(string)
	service, _ := delivery.Headers["Service"].(string)
	endpoint, _ := delivery.Headers["Endpoint"].(string)
	accessToken, _ := delivery.Headers["Access-Token"].(string)
	parentRequestID, _ := delivery.Headers["Parent-Request-ID"].(string)
	sessionBytes, _ := delivery.Headers["Session"].(string)

	var (
		session auth.Session
		err     error
	)
	if sessionBytes != "" && s.AuthenticationProvider() != nil {
		session, err = s.AuthenticationProvider().UnmarshalSession([]byte(sessionBytes))
		if err != nil {
			return nil, err
		}
	}

	// @todo if traceID is empty, that indicates a problem that should be logged somewhere
	traceID, _ := delivery.Headers["Trace-ID"].(string)

	return &AMQPRequest{
		s:               s,
		delivery:        delivery,
		id:              delivery.CorrelationId,
		contentType:     contentType,
		contentEncoding: contentEncoding,
		service:         service,
		endpoint:        endpoint,
		accessToken:     accessToken,
		parentRequestID: parentRequestID,
		traceID:         traceID,
		session:         session,
	}, nil
}

// RabbitMQ / AMQP fields

// Id specifies the request's unique identifier
func (r *AMQPRequest) Id() string {
	return r.id
}

// ContentType of this request - eg. json, protobuf
func (r *AMQPRequest) ContentType() string {
	return r.contentType
}

// ContentEncoding of this request - eg. request, response, error
func (r *AMQPRequest) ContentEncoding() string {
	return r.contentEncoding
}

// Service which should process this request
func (r *AMQPRequest) Service() string {
	return r.service
}

// Endpoint which should process this request
func (r *AMQPRequest) Endpoint() string {
	return r.endpoint
}

// Payload is the raw delivery body
func (r *AMQPRequest) Payload() []byte {
	return r.delivery.Body
}

// Body of the request is populated if the payload was successfully
// unmarshalled to a protobuf message by our helper methods
func (r *AMQPRequest) Body() interface{} {
	return r.body
}

// SetBody of the request to a decoded payload
func (r *AMQPRequest) SetBody(body interface{}) {
	r.body = body
}

// CorrelationID allows our services to correlate requests and responses
func (r *AMQPRequest) CorrelationID() string {
	return r.id
}

// ReplyTo specifies the client to reply to
func (r *AMQPRequest) ReplyTo() string {
	return r.delivery.ReplyTo
}

// RoutingKey of the request used to route the message to this service
func (r *AMQPRequest) RoutingKey() string {
	return r.delivery.RoutingKey
}

// Interface returns the raw AMQP delivery
func (r *AMQPRequest) Interface() interface{} {
	return r.delivery
}

// Session returns the session associated with this request and recovers it
// using AuthenticationProvider's RecoverSession if not set
func (r *AMQPRequest) Session() (auth.Session, error) {
	if r.session != nil {
		log.Debugf("req.Session() returning memoized session %+v", r.session)
		return r.session, nil
	}
	if r.AccessToken() == "" {
		return nil, nil
	}
	authProvider := r.Server().AuthenticationProvider()
	if authProvider == nil {
		log.Warnf("Server doesn't have an AuthenticationProvider, returning nil session")
		return nil, nil
	}
	session, err := authProvider.RecoverSession(r, r.AccessToken())
	if err != nil {
		return nil, err
	}
	r.session = session
	return r.session, nil
}

// SetSession information into the request
func (r *AMQPRequest) SetSession(s auth.Session) {
	r.session = s
}

// HasRecoveredSession returns true if the session was previously successfully
// recovered from the access token
func (r *AMQPRequest) HasRecoveredSession() bool {
	return r.session != nil
}

// Server returns the server which is processing this request
func (r *AMQPRequest) Server() Server {
	return r.s
}

// AccessToken returns the authentication token sent with this request
func (r *AMQPRequest) AccessToken() string {
	return r.accessToken
}

// TraceID is the trace id of this request. It should never be unset
func (r *AMQPRequest) TraceID() string {
	return r.traceID
}

// ParentRequestID is the ID of the parent request, if any
func (r *AMQPRequest) ParentRequestID() string {
	return r.parentRequestID
}
