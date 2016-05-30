package httpsvc

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// dialer is a custom net.Dialer implementation that only allows a certain number of connections to a host. It is used
// to limit peak connection concurrency in the HTTP client implementation.
type dialer struct {
	net.Dialer
	tokens          map[string]chan struct{}
	tokensM         sync.RWMutex
	MaxConnsPerHost int
}

func (d *dialer) token(network, address string) chan struct{} {
	key := network + address
	d.tokensM.RLock()
	c, ok := d.tokens[key]
	d.tokensM.RUnlock()
	if ok {
		return c
	}
	d.tokensM.Lock()
	defer d.tokensM.Unlock()
	if d.tokens == nil {
		d.tokens = make(map[string]chan struct{}, 10)
	}
	c, ok = d.tokens[key]
	if !ok {
		c = make(chan struct{}, d.MaxConnsPerHost)
		for len(c) < cap(c) {
			c <- struct{}{}
		}
		d.tokens[key] = c
	}
	return c
}

func (d *dialer) Dial(network, address string) (net.Conn, error) {
	t := time.NewTimer(d.Timeout / 2)
	defer t.Stop()
	token := d.token(network, address)
	select {
	case <-token:
		conn, err := d.Dialer.Dial(network, address)
		return &trackedConn{
			Conn:  conn,
			token: token}, err
	case <-t.C:
		return nil, fmt.Errorf("Timed out waiting for connection token")
	}
}

type trackedConn struct {
	net.Conn
	token chan<- struct{}
}

func (t *trackedConn) Close() error {
	err := t.Conn.Close()
	select {
	case t.token <- struct{}{}:
	default:
	}
	return err
}
