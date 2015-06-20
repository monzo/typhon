package message

import (
	"sync"
)

type message struct {
	sync.RWMutex
	id       string
	payload  []byte
	body     interface{}
	service  string
	endpoint string
	headers  map[string]string
}

// Message implementation

func (m *message) Id() string {
	m.RLock()
	defer m.RUnlock()
	return m.id
}

func (m *message) Payload() []byte {
	m.RLock()
	defer m.RUnlock()
	return m.payload
}

func (m *message) Body() interface{} {
	m.RLock()
	defer m.RUnlock()
	return m.body
}

func (m *message) Service() string {
	m.RLock()
	defer m.RUnlock()
	return m.service
}

func (m *message) Endpoint() string {
	m.RLock()
	defer m.RUnlock()
	return m.endpoint
}

func (m *message) headersCopy() map[string]string {
	// Callers must ensure they hold a read lock before invoking this method
	result := make(map[string]string, len(m.headers))
	for k, v := range m.headers {
		result[k] = v
	}
	return result
}

func (m *message) Headers() map[string]string {
	m.RLock()
	defer m.RUnlock()
	return m.headersCopy()
}

func (m *message) copy() message {
	// Callers must ensure they hold a read lock before invoking this method
	return message{
		id:       m.id,
		payload:  m.payload,
		body:     m.body,
		service:  m.service,
		endpoint: m.endpoint,
		headers:  m.headersCopy(),
	}
}

func (m *message) SetId(id string) {
	m.Lock()
	defer m.Unlock()
	m.id = id
}

func (m *message) SetPayload(p []byte) {
	m.Lock()
	defer m.Unlock()
	m.payload = p
}

func (m *message) SetBody(b interface{}) {
	m.Lock()
	defer m.Unlock()
	m.body = b
}

func (m *message) SetService(s string) {
	m.Lock()
	defer m.Unlock()
	m.service = s
}

func (m *message) SetEndpoint(e string) {
	m.Lock()
	defer m.Unlock()
	m.endpoint = e
}

func (m *message) SetHeader(k, v string) {
	m.Lock()
	defer m.Unlock()
	m.headers = m.headersCopy()
	m.headers[k] = v
}

func (m *message) UnsetHeader(k string) {
	m.Lock()
	defer m.Unlock()
	if _, ok := m.headers[k]; !ok { // Header isn't set; no need to copy
		return
	}
	m.headers = m.headersCopy()
	delete(m.headers, k)
}

func (m *message) SetHeaders(h map[string]string) {
	m.Lock()
	defer m.Unlock()
	m.headers = h
}
