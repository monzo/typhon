package httpcompat

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"gopkg.in/tomb.v2"

	"github.com/mondough/typhon/http"
	"github.com/mondough/typhon/message"
	oldtrans "github.com/mondough/typhon/transport"
)

type transportUpgrader struct {
	trans                httpsvc.Transport
	tomb                 *tomb.Tomb
	inflightReqsM        sync.RWMutex                       // protects inflightReqs
	inflightReqs         map[string]chan<- httpsvc.Response // holds requests
	reqz                 *uint64                            // accessed atomically
	registeredListenersM sync.Mutex                         // protects registeredListeners
	registeredListeners  map[string]struct{}                // used to provide "already listening" behaviour
}

// New2OldTransport takes a new-fangled HTTP transport and wraps it in an interface that is compatible with the
// old-fashioned Transport API.
//
// @DEPRECATED: This is purely to ease the transition and will go away.
func New2OldTransport(t httpsvc.Transport) oldtrans.Transport {
	zero := uint64(0)
	u := &transportUpgrader{
		trans:               t,
		tomb:                new(tomb.Tomb),
		reqz:                &zero,
		inflightReqs:        make(map[string]chan<- httpsvc.Response, 500),
		registeredListeners: make(map[string]struct{}, 1)}
	u.run()
	return u
}

func (t *transportUpgrader) run() {
	t.tomb.Go(func() error {
		<-t.tomb.Dying()
		t.trans.Close(30 * time.Second)
		return nil
	})
}

func (t *transportUpgrader) reqId() string {
	i := atomic.AddUint64(t.reqz, 1)
	return strconv.FormatUint(i, 16)
}

func (t *transportUpgrader) Tomb() *tomb.Tomb {
	return t.tomb
}

func (t *transportUpgrader) Ready() <-chan struct{} {
	c := make(chan struct{})
	close(c)
	return c
}

func (t *transportUpgrader) Listen(name string, inboundChan chan<- message.Request) error {
	t.registeredListenersM.Lock()
	defer t.registeredListenersM.Unlock()
	if _, ok := t.registeredListeners[name]; ok {
		return oldtrans.ErrAlreadyListening
	}
	t.registeredListeners[name] = struct{}{}

	svc := func(req httpsvc.Request) httpsvc.Response {
		id := t.reqId()
		rspChan := make(chan httpsvc.Response, 1)
		t.inflightReqsM.Lock()
		t.inflightReqs[id] = rspChan
		t.inflightReqsM.Unlock()
		defer func() {
			t.inflightReqsM.Lock()
			delete(t.inflightReqs, id)
			t.inflightReqsM.Unlock()
		}()
		oldReq := new2OldRequest(req)
		oldReq.SetId(id)
		inboundChan <- oldReq
		return <-rspChan
	}
	return t.trans.Listen(name, svc)
}

func (t *transportUpgrader) StopListening(name string) bool {
	t.trans.Unlisten(name)
	t.registeredListenersM.Lock()
	delete(t.registeredListeners, name)
	t.registeredListenersM.Unlock()
	return true
}

func (t *transportUpgrader) Respond(oldReq message.Request, oldRsp message.Response) error {
	t.inflightReqsM.RLock()
	rspChan, ok := t.inflightReqs[oldReq.Id()]
	t.inflightReqsM.RUnlock()
	if ok {
		rspChan <- old2NewResponse(oldRsp)
		return nil
	}
	return fmt.Errorf("No matching response channel %s", oldReq.Id())
}

func (t *transportUpgrader) Send(req message.Request, timeout time.Duration) (message.Response, error) {
	if req.Id() == "" {
		req.SetId(t.reqId())
	}
	svc := httpsvc.
		Service(t.trans.Send).
		Filtered(httpsvc.TimeoutFilter(timeout))
	return new2OldResponse(svc(old2NewRequest(req)))
}
