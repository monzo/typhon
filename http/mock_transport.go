package httpsvc

import (
	"sync"
	"time"
)

type mockTransport struct {
	servicesM sync.RWMutex       // protects services
	services  map[string]Service // maps service names to services
}

func MockTransport() Transport {
	return &mockTransport{
		services: make(map[string]Service, 1)}
}

func (t *mockTransport) Listen(name string, svc Service) error {
	t.servicesM.Lock()
	defer t.servicesM.Unlock()
	t.services[name] = svc
	return nil
}

func (t *mockTransport) Unlisten(name string) {
	t.servicesM.Lock()
	delete(t.services, name)
	t.servicesM.Unlock()
}

func (t *mockTransport) Close(timeout time.Duration) {
	t.servicesM.Lock()
	t.services = make(map[string]Service, 1)
	t.servicesM.Unlock()
}

func (t *mockTransport) Send(req Request) Response {
	t.servicesM.RLock()
	svc, ok := t.services[req.Host]
	t.servicesM.RUnlock()
	if ok {
		return svc(req)
	}
	panic("not handled")
}
