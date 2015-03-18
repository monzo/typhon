package errors

type Error struct {
	Code           int
	Message        string
	PublicContext  map[string]string
	PrivateContext map[string]string
}

// Default error codes. Each of these has their own constructor for convenience.
// You can use any integer as a code, just use the `New` method.
const (
	ErrUnknown         = 0
	ErrInternalService = 1
	ErrBadRequest      = 2
	ErrBadResponse     = 3
	ErrForbidden       = 4
	ErrUnauthorized    = 5
	ErrNotFound        = 6
	ErrTimeout         = 7
)

// Error returns a string message of the error
// This means the Error implements the error interface
func (p *Error) Error() string {
	if p == nil {
		return "FATAL: Nil error!"
	}
	return p.Message
}

func New(code int, message string, context ...map[string]string) *Error {
	return errorFactory(code, message, context...)
}

// Wrap takes any error interface and wraps it into an Error.
// This is useful because an Error contains lots of useful goodies, like the stacktrace of the error.
// If `err` is already an `Error` the passed public and private contexts (if any) will be merged into `err`
func Wrap(err error, context ...map[string]string) *Error {
	wrappedErr, ok := err.(*Error)
	if !ok {
		wrappedErr = errorFactory(ErrInternalService, err.Error(), context...)
	} else {
		if len(context) >= 1 {
			mergeMaps(wrappedErr.PublicContext, context[0])
		}
		if len(context) >= 2 {
			mergeMaps(wrappedErr.PrivateContext, context[1])
		}
	}
	return wrappedErr
}

// InternalService creates a new error to represent an internal service error.
// Only use internal service error if we know very little about the error. Most
// internal service errors will come from `Wrap`ing a vanilla `error` interface
func InternalService(message string, context ...map[string]string) *Error {
	return errorFactory(ErrInternalService, message, context...)
}

// BadRequest creates a new error to represent an error caused by the client sending
// an invalid request. This is non-retryable unless the request is modified.
func BadRequest(message string, context ...map[string]string) *Error {
	return errorFactory(ErrBadRequest, message, context...)
}

// BadResponse creates a new error representing a failure to response with a valid response
// Examples of this would be a handler returning an invalid message format
func BadResponse(message string, context ...map[string]string) *Error {
	return errorFactory(ErrBadResponse, message, context...)
}

// Timeout creates a new error representing a timeout from client to server
func Timeout(message string, context ...map[string]string) *Error {
	return errorFactory(ErrTimeout, message, context...)
}

// NotFound creates a new error representing a resource that cannot be found. In some
// cases this is not an error, and would be better represented by a zero length slice of elements
func NotFound(message string, context ...map[string]string) *Error {
	return errorFactory(ErrNotFound, message, context...)
}

// Forbidden creates a new error representing a resource that cannot be accessed with
// the current authorisation credentials. The user may need authorising, or if authorised,
// may not be permitted to perform this action
func Forbidden(message string, context ...map[string]string) *Error {
	return errorFactory(ErrForbidden, message, context...)
}

// Unauthorized creates a new error indicating that authentication is required,
// but has either failed or not been provided.
func Unauthorized(message string, context ...map[string]string) *Error {
	return errorFactory(ErrUnauthorized, message, context...)
}

// errorConstructor returns a `ServiceError` of the specified code. The main work
// consists of managing the map[string]string at the end of the arguments list.
// In practice we only ever pass in two: the first one is public and will be sent to the client,
// the second one is private and can contain internal information that is useful for debugging
// @todo get a stack trace and stick it in the private context
// @todo should we do some meta-error-handling here and complain if the code is "" for example?
func errorFactory(code int, message string, context ...map[string]string) *Error {
	err := &Error{
		Code:           code,
		Message:        message,
		PrivateContext: map[string]string{},
		PublicContext:  map[string]string{},
	}
	// The first context map is the PublicContext
	if len(context) > 0 {
		err.PublicContext = context[0]
	}
	// The second context map is the privateContext
	if len(context) > 1 {
		err.PrivateContext = context[1]
	}
	// ... ignore all remaining map[string]string
	return err
}

func mergeMaps(dest, source map[string]string) {
	if dest == nil || source == nil {
		return
	}
	for key, val := range source {
		dest[key] = val
	}
}
