package transport

type Transport interface {
	Init() chan bool
	Consume(serverName string) <-chan *Request
}
