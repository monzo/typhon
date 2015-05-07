package auth

import (
	"time"

	"golang.org/x/net/context"
)

// AuthenticationProvider provides helper methods to convert tokens to sessions
// using our own internal authorization services
type AuthenticationProvider interface {
	// MarshalSession into wire format for transmission between services
	MarshalSession(s Session) ([]byte, error)
	// UnmarshalSession from wire format used during transmission between services
	UnmarshalSession(b []byte) (Session, error)

	// RecoverSession from a given access token, converting this into a session
	RecoverSession(ctx context.Context, accessToken string) (Session, error)
}

// Session represents an OAuth access token along with expiry information,
// user and client information
type Session interface {
	AccessToken() string
	RefreshToken() string
	Expiry() time.Time
	// @todo add Signature() string

	User() User
	Client() Client
}

// Authorizer provides an interface to validate session authorization
// for access to resources, eg. oauth scopes, or other access control
type Authorizer func(ctx context.Context) error

// User represents the resource owner ie. an end-user of the application
type User interface {
	ID() string
	Scopes() []string
}

// Client represents the application making a request on behalf of a User
type Client interface {
	ID() string
	Scopes() []string
}
