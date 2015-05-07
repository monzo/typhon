package server

import log "github.com/cihub/seelog"

// authenticateEndpointAccess for a given request
func authenticateEndpointAccess(ctx Request, e *Endpoint) error {

	// We also need an Authentication Provider to actually handle authentication tasks
	ap := e.server.AuthenticationProvider()
	if ap == nil {
		log.Debugf("No authentication provider configured, skipping authentication")
		return nil
	}

	// Recover session from Authentication Provider
	// @todo this should be moved into the authorizer so that
	// it is lazily evaluated
	session, err := ap.RecoverSession(ctx, ctx.Session().AccessToken())
	if err != nil {
		return err
	}

	return e.Authorizer(ctx, session)
}
