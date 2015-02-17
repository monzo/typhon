package server

type Server interface {
	Init()
	Run()

	RegisterEndpoint(endpoint Endpoint)
	DeregisterEndpoint(pattern string)
}

var DefaultServer Server

// RegisterEndpoint with the DefaultServer
func RegisterEndpoint(endpoint Endpoint) {
	DefaultServer.RegisterEndpoint(endpoint)
}

// Run the DefaultServer
func Run() {
	DefaultServer.Run()
}
