package server

import (
	log "github.com/cihub/seelog"
)

// authenticateEndpointAccess for a given request
func authenticateEndpointAccess(e *Endpoint, req Request) error {

	// First check if we need authentication on this endpoint
	if e.Authorizer == nil {
		log.Debugf("No authorizer set for endpoint %s, skipping authentication", e.Name)
		return nil
	}

	// We also need an Authentication Provider to actually handle authentication tasks
	ap := e.server.AuthenticationProvider()
	if ap == nil {
		log.Debugf("No authentication provider configured, skipping authentication")
		return nil
	}

	// Recover credentials from Authentication Provider
	// @todo this should be moved into the authorizer so that
	// it is lazily evaluated
	creds, err := ap.RecoverCredentials(req, req.AccessToken())
	if err != nil {
		return err
	}

	return e.Authorizer(req, creds)
}
