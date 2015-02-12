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

func NewServer(name string, tp transport.Transport) *Server {
	return &Server{
		Name:      name,
		Transport: tp,
		registry:  registry.NewRegistry(),
	}
}

func (s *Server) Init() {
	select {
	case <-s.Transport.Init():
		log.Info("[Server] Connected to transport")
	case <-time.After(10 * time.Second):
		log.Critical("[Server] Failed to connect to transport")
		os.Exit(1)
	}
}
