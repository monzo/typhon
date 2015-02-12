package transport

type Transport interface {
	Init() chan bool
}
