package transport

import (
	"time"

	"github.com/mondough/terrors"
	"github.com/mondough/typhon/message"
	"gopkg.in/tomb.v2"
)

var (
	// ErrAlreadyListening indicates a listener channel is already active for a given service
	ErrAlreadyListening = terrors.InternalService("", "Listener already registered for service", nil)
	// ErrTimeout indicates a timeout was exceeded
	ErrTimeout = terrors.Timeout("", "Timed out", nil)
)

// A Transport provides a persistent interface to a transport layer. It is capable of sending and receiving Messages
// on behalf of multiple services in parallel.
type Transport interface {
	// A Tomb tracking the lifecycle of the Transport.
	Tomb() *tomb.Tomb
	// Ready vends a channel to wait on until the Transport is ready for use. Note that this will block indefinitely if
	// the Transport never reaches readiness. When ready, the channel is closed (so it's safe to listen in many
	// goroutines)
	Ready() <-chan struct{}
	// Listen for requests destined for a specific service, forwarding them down the passed channel. If another listener
	// is already listening, returns ErrAlreadyListening.
	Listen(serviceName string, inboundChan chan<- message.Request) error
	// StopListening terminates a listener for the passed service, returning whether successful.
	StopListening(serviceName string) bool
	// Respond sends a response to a Request.
	Respond(request message.Request, response message.Response) error
	// Send transmits an outbound request to another service, waiting for a response, a timeout, or another error.
	Send(req message.Request, timeout time.Duration) (message.Response, error)
}
