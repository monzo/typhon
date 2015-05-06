package auth

import (
	log "github.com/cihub/seelog"
	"golang.org/x/net/context"
)

// DefaultAuthorizer is an authorizer used on endpoint registration if the endpoint
// does not have an authorizer specified. It is STRONGLY recommended that this be replaced
// by a significantly more secure authorizer, preventing unintended access to endpoints
var DefaultAuthorizer = None()

// None authorizer requires no authorization and is therefore accessible to all callers
func None() func(ctx context.Context, s Session) error {
	return func(ctx context.Context, s Session) error {
		log.Debugf("No auth required for endpoint as the None authorizer is being used")
		return nil
	}
}
