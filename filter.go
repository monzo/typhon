package typhon

// Filter functions compose with Services to modify their behaviour. They might change a service's input or output, or
// elect not to call the underlying service at all.
//
// These are typically useful to encapsulate common logic that is shared among multiple Services. Authentication,
// authorisation, rate limiting, and tracing are good examples.
type Filter func(Request, Service) Response
