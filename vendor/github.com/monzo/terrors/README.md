`terrors`
=========

[![Build Status](https://travis-ci.org/monzo/terrors.svg)](https://travis-ci.org/monzo/terrors)
[![GoDoc](https://godoc.org/github.com/monzo/terrors?status.svg)](https://godoc.org/github.com/monzo/terrors)

Terrors is a package for wrapping Golang errors. Terrors provides additional
context to an error, such as an error code and a stack trace.

Terrors is built and used at [Monzo](https://monzo.com/).

## Usage

Terrors can be used to wrap any object that satisfies the error interface:

```go
terr := terrors.Wrap(err, map[string]string{"context": "my_context"})
```
Terrors can be instantiated directly:

```go
err := terrors.New("not_found", "object not found", map[string]string{
	"context": "my_context"
})
```

Terrors offers built-in functions for instantiating `Error`s with common codes:

```go
err := terrors.NotFound("config_file", "config file not found", map[string]string{
	"context": my_context
})
```

Terrors provides functions for matching specific `Error`s:

```go
err := NotFound("handler_missing", "Handler not found", nil)
fmt.Println(Matches(err, "not_found.handler_missing")) // true
```

## API

Full API documentation can be found on
[godoc](https://godoc.org/github.com/monzo/terrors)

## Install

```
$ go get -u github.com/monzo/terrors
```

## License

Terrors is licenced under the MIT License
