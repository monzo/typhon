package server

// Response defines an interface that all handler responses must satisfy
type Response interface {
	Encode() ([]byte, error)
}
