package message

// A Marshaler knows how to marshal a Message's Body into Payload bytes ready for transport.
type Marshaler interface {
	MarshalBody(msg Message) error
}

// An Unmarshaler knows how to take a Message's raw Payload and unmarshal it into the Body. If the message's Body is
// non-nil, it should unmarshal into the existing object pointed to.
type Unmarshaler interface {
	UnmarshalPayload(msg Message) error
}

// A Message represents a discrete communication between two services (ie. a Request or a Response).
type Message interface {
	// An identifier for a given communication; this does not have to be unique. Indeed, Requests share their Id with
	// a correlated Response.
	Id() string
	// Payload of raw (body, not header) bytes sent over the transport.
	Payload() []byte
	// Body contains the unmarshalled Payload (and may be nil).
	Body() interface{}
	// The destination service.
	Service() string
	// The destination endpoint.
	Endpoint() string
	// The originating service.
	OriginService() string
	// The originating endpoint.
	OriginEndpoint() string
	// Headers returns a map of header keys and their values. Mutating the map will have no effect on the Request.
	Headers() map[string]string

	// SetId sets the Id
	SetId(id string)
	// SetPayload sets the raw bytes sent over the transport; an Unmarshaler invocation should usually follow.
	SetPayload(payload []byte)
	// SetBody sets the unmarshalled body; a Marshaler invocation should usually follow.
	SetBody(body interface{})
	// SetService sets the destination service.
	SetService(service string)
	// SetEndpoint sets the destination endpoint.
	SetEndpoint(endpoint string)
	// SetOriginService sets the originating service.
	SetOriginService(service string)
	// SetOriginEndpoint sets the originating endpoint.
	SetOriginEndpoint(endpoint string)
	// SetHeader sets the value of a given header key.
	SetHeader(key, value string)
	// UnsetHeader removes a given header. Removing a nonexistent header is a no-op.
	UnsetHeader(key string)
	// SetHeaders sets (ie. overwrites) all headers as a batch operation
	SetHeaders(headers map[string]string)
}

// A Request is a representation of a service call (inbound or outbound).
type Request interface {
	Message

	// Copy returns an identical, shallow copy of the Request.
	Copy() Request
}

// A Response is a correlated reply to a Request (inbound or outbound.)
type Response interface {
	Message

	// Copy returns an identical, shallow copy of the Response.
	Copy() Response
}
