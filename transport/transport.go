package transport

type Transport interface {
	Connect() chan *Transport
}
