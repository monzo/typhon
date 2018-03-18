package typhon

// Filter functions compose with Services to modify their observed behaviour. They might change a service's input or
// output, or elect not to call the underlying service at all.
type Filter func(Request, Service) Response
