package transport

type Request interface {
	Body() []byte
	Interface() interface{}
	RoutingKey() string
	ReplyTo() string
}
