package server

import "github.com/golang/protobuf/proto"

// Response defines an interface that all handler responses must satisfy
type Response interface {
	Encode() ([]byte, error)
}

// Various Response type implementations

// NewProtoResponse creates a new Response from a protobuf message
func NewProtoResponse(p proto.Message) Response {
	return &ProtoResponse{
		Pb: p,
	}
}

// ProtoResponse represents a protobuf message used as a response
type ProtoResponse struct {
	// this is only public so that endpoint.go can log the response as it passes through
	// @todo make the Response interface less opaque to allow for logging or just couple the whole server package to proto
	Pb proto.Message
}

// Encode the protobuf message to bytes for transmission
func (p *ProtoResponse) Encode() ([]byte, error) {
	if p == nil || p.Pb == nil {
		return []byte{}, nil
	}

	return proto.Marshal(p.Pb)
}
