package server

import (
	"os"
	"time"

	log "github.com/cihub/seelog"
	"github.com/vinceprignano/bunny/server/registry"
	"github.com/vinceprignano/bunny/transport"
)

type Server struct {
	Name      string
	Transport transport.Transport
	registry  *registry.Registry
}

func (s *Server) Init() {
	select {
	case s.Transport = <-s.Transport.Connect():
		log.Info("[Server] Connected to transport")
	case <-time.After(10 * time.Second):
		log.Critical("[Server] Failed to connect to transport")
		os.Exit(1)
	}
}
