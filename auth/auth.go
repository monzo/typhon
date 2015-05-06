package auth

import (
	"time"

	"github.com/b2aio/typhon/server"
	"golang.org/x/net/context"
)

// AuthenticationProvider provides helper methods to convert tokens to credentials
// using our own internal authorization services
type AuthenticationProvider interface {
	// MarshalCredentials into wire format for transmission between services
	MarshalCredentials(c Credentials) ([]byte, error)
	// UnmarshalCredentials from wire format used during transmission between services
	UnmarshalCredentials(b []byte) (Credentials, error)

	// RecoverSession from a given access token, converting this into a set of credentials
	RecoverCredentials(ctx context.Context, accessToken string) (Credentials, error)
}

// Credentials represent an OAuth access token along with expiry information
// user and client information
type Credentials interface {
	AccessToken() string
	RefreshToken() string
	Expiry() time.Time
	Scopes() []string // aggregated scope information from a combination of the user and client scopes
	User() User
	Client() Client
}

// Authorizer provides an interface to validate authorization credentials
// for access to resources, eg. oauth scopes, or other access control
type Authorizer func(ctx server.Request) error

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
