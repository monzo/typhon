package server

// Server is an interface that all servers must implement
// so that we can register endpoints, and serve requests
type Server interface {
	Init(*Config)
	Run()
	NotifyConnected() chan bool

	Name() string
	Description() string

	RegisterEndpoint(endpoint Endpoint)
	DeregisterEndpoint(pattern string)
}

// DefaultServer stores a default implementation, for simple usage
var DefaultServer Server = NewAMQPServer()

// Init our DefaultServer with a Config
func Init(c *Config) {
	DefaultServer.Init(c)
}

// RegisterEndpoint with the DefaultServer
func RegisterEndpoint(endpoint Endpoint) {
	DefaultServer.RegisterEndpoint(endpoint)
}

// Run the DefaultServer
func Run() {
	DefaultServer.Run()
}

// Config defines the config a server needs to start up, and serve requests
type Config struct {
	Name        string
	Description string
}
