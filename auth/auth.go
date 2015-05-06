package auth

import (
	"time"

	"golang.org/x/net/context"
)

// AuthenticationProvider provides helper methods to convert tokens to sessions
// using our own internal authorization services
type AuthenticationProvider interface {
	// RecoverSession from a given access token, converting this into a set of credentials
	RecoverCredentials(ctx context.Context, accessToken string) (Credentials, error)
}

// Credentials
type Credentials interface {
	AccessToken() string
	RefreshToken() string
	Expiry() time.Time
	Scopes() []string
}

// Authorizer provides an interface to validate authorization credentials
// for access to resources, eg. oauth scopes, or other access control
type Authorizer func(ctx context.Context, creds Credentials) error
