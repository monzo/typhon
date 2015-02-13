package transport

type Transport interface {
	Init() chan bool
	Consume(serverName string) <-chan Request
	PublishFromRequest(req Request, body []byte, err error)
	Publish(routingKey string, body []byte)
}
