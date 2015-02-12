package server

import "github.com/vinceprignano/bunny/server/endpoint"

type Server struct {
	endpoints map[string]endpoint.Endpoint
}
