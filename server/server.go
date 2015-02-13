package server

import (
	"fmt"
	"os"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"github.com/vinceprignano/bunny/transport"
)

type Server struct {
	Name      string
	Transport transport.Transport
	registry  *Registry
}

func NewServer(name string, tp transport.Transport) *Server {
	return &Server{
		Name:      name,
		Transport: tp,
		registry:  NewRegistry(),
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

func (s *Server) RegisterEndpoint(endpoint Endpoint) {
	s.registry.RegisterEndpoint(endpoint)
}

func (s *Server) Run() {
	log.Infof("[Server] Listening for deliveries")

	// Range over deliveries from channel
	// This blocks until the channel closes
	for req := range s.Transport.Consume(s.Name) {
		log.Info("[Server] Recevied new request")
		go s.handleRequest(req)
	}

	log.Infof("Exiting")
	log.Flush()
}

func (s *Server) handleRequest(req transport.Request) {
	endpointName := strings.Replace(req.RoutingKey(), fmt.Sprintf("%s.", s.Name), "", -1)
	endpoint := s.registry.GetEndpoint(endpointName)
	if endpoint == nil {
		log.Error("[Server] Endpoint not found, cannot handle request")
		return
	}
	rsp, err := endpoint.HandleRequest(req)
	s.Transport.PublishFromRequest(req, rsp, err)
}
