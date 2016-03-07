package mock

import (
	"fmt"
	"gopkg.in/tomb.v2"
	"sync"
	"time"

	log "github.com/mondough/slog"
	uuid "github.com/nu7hatch/gouuid"
	"golang.org/x/net/context"

	"github.com/mondough/typhon/message"
	"github.com/mondough/typhon/transport"
)

const (
	timeout = time.Second
)

type mockListener struct {
	tomb    *tomb.Tomb
	reqChan chan<- message.Request
}

type mockTransport struct {
	sync.RWMutex
	tomb         *tomb.Tomb
	listeners    map[string]*mockListener
	ready        chan struct{}                      // closed immediately for this transport
	inflightReqs map[string]chan<- message.Response // correlation id: response chan
}

func (mt *mockTransport) run() error {
	close(mt.ready)
	<-mt.tomb.Dying()
	mt.killListeners()
	return tomb.ErrDying
}

func (t *mockTransport) Ready() <-chan struct{} {
	return t.ready
}

func (t *mockTransport) Tomb() *tomb.Tomb {
	return t.tomb
}

func (t *mockTransport) killListeners() {
	t.RLock()
	ts := make([]*tomb.Tomb, 0, len(t.listeners))
	for _, l := range t.listeners {
		l.tomb.Killf("Listeners killed")
		ts = append(ts, l.tomb)
	}
	t.RUnlock()
	for _, t := range ts {
		t.Wait()
	}
}

func (t *mockTransport) StopListening(serviceName string) bool {
	t.RLock()
	l, ok := t.listeners[serviceName]
	if ok {
		l.tomb.Killf("Stopped listening")
	}
	t.RUnlock()
	if ok {
		l.tomb.Wait()
		return true
	}
	return false
}

func (t *mockTransport) Listen(serviceName string, rc chan<- message.Request) error {
	ctx := context.Background()
	l := &mockListener{
		tomb:    new(tomb.Tomb),
		reqChan: rc,
	}
	t.Lock()
	if _, ok := t.listeners[serviceName]; ok {
		t.Unlock()
		return transport.ErrAlreadyListening
	} else {
		t.listeners[serviceName] = l
		t.Unlock()
	}

	stop := func() {
		log.Debug(ctx, "[Typhon:MockTransport] Listener %s stopped", serviceName)
		t.Lock()
		defer t.Unlock()
		delete(t.listeners, serviceName)
	}

	// Wait here, rather than inside the goroutine, because we want to be able to report an (un)successful connection
	tm := l.tomb
	select {
	case <-t.tomb.Dying():
		stop()
		return tomb.ErrDying
	case <-tm.Dying():
		stop()
		return tomb.ErrDying
	case <-time.After(timeout):
		stop()
		return transport.ErrTimeout
	case <-t.Ready():
		log.Debug(ctx, "[Typhon:MockTransport] Listener %s started", serviceName)
	}

	tm.Go(func() error {
		defer stop()
		<-tm.Dying()
		return tomb.ErrDying
	})
	return nil
}

func (t *mockTransport) Send(req message.Request, timeout time.Duration) (message.Response, error) {
	ctx := context.Background()
	id := req.Id()
	if id == "" {
		_uuid, err := uuid.NewV4()
		if err != nil {
			log.Error(ctx, "[Typhon:MockTransport] Failed to generate request uuid: %v", err)
			return nil, err
		}
		req.SetId(_uuid.String())
	}

	// Make a copy of the response that does not preserve the Body (this is not preserved over the wire)
	req = req.Copy()
	req.SetBody(nil)

	t.RLock()
	l, ok := t.listeners[req.Service()]
	t.RUnlock()
	if ok {
		responseChan := make(chan message.Response, 1)
		t.Lock()
		t.inflightReqs[req.Id()] = responseChan
		t.Unlock()
		defer func() {
			t.Lock()
			delete(t.inflightReqs, req.Id())
			t.Unlock()
		}()

		timer := time.NewTimer(timeout)
		defer timer.Stop()
		select {
		case <-timer.C:
			log.Debug(ctx, "[Typhon:MockTransport] Timed out after %v waiting for delivery of \"%s\"", timeout, req.Id())
			return nil, transport.ErrTimeout
		case l.reqChan <- req:
		}

		select {
		case rsp := <-responseChan:
			return rsp, nil
		case <-timer.C:
			log.Debug(ctx, "[Typhon:MockTransport] Timed out after %v waiting for response to \"%s\"", timeout, req.Id())
			return nil, transport.ErrTimeout
		}
	}

	return nil, transport.ErrTimeout // Don't bother waiting artificially
}

func (t *mockTransport) Respond(req message.Request, rsp message.Response) error {
	t.RLock()
	rspChan, ok := t.inflightReqs[req.Id()]
	t.RUnlock()

	// Make a copy of the response that does not preserve the Body (this is not preserved over the wire)
	rsp = rsp.Copy()
	rsp.SetBody(nil)

	if ok {
		select {
		case rspChan <- rsp:
			return nil
		case <-time.After(timeout):
			return transport.ErrTimeout
		}
	}
	return fmt.Errorf("No correlated request \"%s\" is in-flight", req.Id())
}

func NewTransport() transport.Transport {
	trans := &mockTransport{
		tomb:         new(tomb.Tomb),
		listeners:    make(map[string]*mockListener),
		ready:        make(chan struct{}),
		inflightReqs: make(map[string]chan<- message.Response),
	}
	trans.tomb.Go(trans.run)
	return trans
}
