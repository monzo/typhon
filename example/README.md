# Example Service

This is an example service using a `client` and `server`

The main program is located in `main.go` which sets up the client and server, along with registering our example endpoint.

The endpoint _handler_ is located in the `handler` package

    .
    └── handler
        └── hello.go

Protocol Buffers are used for message formats, and are located in the `proto` directory, with a folder per endpoint. The `hello.proto` file has a `Request` type and a `Response` type, which are the input and output message envelopes of the endpoint.

    .
    └── proto
        └── hello
            ├── hello.pb.go
            └── hello.proto

