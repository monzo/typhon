package errors

// Error represents all errors which can be passed between services
type Error interface {
	Code() string
	Error() string
	Description() string
	Type() ErrorType
	ClientCode() int
	PublicContext() map[string]string
	PrivateContext() map[string]string
}

// ErrorType is an enumerated type of error
type ErrorType string

const (
	// ErrUnknown indicates an unknown type of error
	// @todo should this just be mapped to an internal server error?
	ErrUnknown = ErrorType("UNKNOWN")

	ErrBadRequest      = ErrorType("BAD_REQUEST")
	ErrBadResponse     = ErrorType("BAD_RESPONSE")
	ErrForbidden       = ErrorType("FORBIDDEN")
	ErrUnauthorized    = ErrorType("UNAUTHORIZED")
	ErrInternalService = ErrorType("INTERNAL_SERVICE")
	ErrNotFound        = ErrorType("NOT_FOUND")
	ErrTimeout         = ErrorType("TIMEOUT")
)

// ServiceError implements the Error interface, and is the internal type we
// use to pass errors between services. The error cannot be directly instantiated,
// and one of the helper methods should be used to construct a specific type of error
type ServiceError struct {
	errorType      ErrorType
	code           string
	description    string
	clientCode     int
	publicContext  map[string]string
	privateContext map[string]string
}

// Code defines a clearly defined inter-service error code
func (p *ServiceError) Code() string {
	if p != nil {
		return p.code
	}

	return ""
}

// NOTE: This will be sent to the client unless `errorType == ErrInternalService`
func (p *ServiceError) Description() string {
	if p != nil {
		return p.description
	}

	return ""
}

// ClientCode returns the error code that the client uses to display.
// There are much fewer client codes than actual error codes. The mapping
// from code to client code is done in `errorFactory` using data from `client_codes.go`
func (p *ServiceError) ClientCode() int {
	if p != nil {
		return p.clientCode
	}

	return DEFAULT_CLIENT_CODE
}

// PublicContext returns a map[string]string of relevant context.
// This context is okay to send to the client. This is useful for
// storing parameters that the client needs for rendering internationalized errors
// such as "Validation of field %s failed (it had value %s)"
func (p *ServiceError) PublicContext() map[string]string {
	if p != nil {
		return p.publicContext
	}

	return nil
}

// PrivateContext returns a map[string]string of relevant context.
// This context is will never be sent to clients. It can be used to store
// lots of debugging information
func (p *ServiceError) PrivateContext() map[string]string {
	if p != nil {
		return p.privateContext
	}

	return nil
}

// Error returns a string description of the error
// This means the ServiceError implements the error interface
func (p *ServiceError) Error() string {
	return p.Description()
}

// Type of error that this error represents
func (p *ServiceError) Type() ErrorType {
	if p != nil && p.errorType != "" {
		return p.errorType
	}

	return ErrUnknown
}

// InternalService creates a new error that represents an error originating within a service
func InternalService(code, description string, context ...map[string]string) Error {
	return errorFactory(ErrInternalService, code, description, context...)
}

// BadRequest creates a new error to represent an error caused by the client sending
// an invalid request. This is non-retryable unless the request is modified.
func BadRequest(code, description string, context ...map[string]string) Error {
	return errorFactory(ErrBadRequest, code, description, context...)
}

// BadResponse creates a new error representing a failure to response with a valid response
// Examples of this would be a handler returning an invalid message format
func BadResponse(code, description string, context ...map[string]string) Error {
	return errorFactory(ErrBadResponse, code, description, context...)
}

// Timeout creates a new error representing a timeout from client to server
func Timeout(code, description string, context ...map[string]string) Error {
	return errorFactory(ErrTimeout, code, description, context...)
}

// NotFound creates a new error representing a resource that cannot be found. In some
// cases this is not an error, and would be better represented by a zero length slice of elements
func NotFound(code, description string, context ...map[string]string) Error {
	return errorFactory(ErrNotFound, code, description, context...)
}

// Forbidden creates a new error representing a resource that cannot be accessed with
// the current authorisation credentials. The user may need authorising, or if authorised,
// may not be permitted to perform this action
func Forbidden(code, description string, context ...map[string]string) Error {
	return errorFactory(ErrForbidden, code, description, context...)
}

// Unauthorized creates a new error indicating that authentication is required,
// but has either failed or not been provided.
func Unauthorized(code, description string, context ...map[string]string) Error {
	return errorFactory(ErrUnauthorized, code, description, context...)
}

// errorConstructor returns a `ServiceError` of the specified type. The main work
// consists of managing the map[string]string at the end of the arguments list.
// In practice we only ever pass in two: the first one is public and will be sent to the client,
// the second one is private and can contain internal information that is useful for debugging
// @todo get a stack trace and stick it in the private context
// @todo should we do some meta-error-handling here and complain if the code is "" for example?
func errorFactory(errorType ErrorType, code, description string, context ...map[string]string) Error {
	err := &ServiceError{
		errorType:      errorType,
		code:           code,
		description:    description,
		privateContext: map[string]string{},
		publicContext:  map[string]string{},
		clientCode:     ClientCodes[code],
	}
	if err.clientCode == 0 {
		err.clientCode = DEFAULT_CLIENT_CODE
	}
	// The first context map is the publicContext
	if len(context) > 0 {
		err.publicContext = context[0]
	}
	// The second context map is the privateContext
	if len(context) > 1 {
		err.privateContext = context[1]
	}
	// ... ignore all remaining map[string]string
	return err
}
