package transport

type Request interface {
	Body() []byte
	Interface() interface{}
}
