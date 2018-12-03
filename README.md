# Typhon 🐲

[![Build Status](https://travis-ci.org/monzo/typhon.svg?branch=master)](https://travis-ci.org/monzo/typhon)
[![GoDoc](https://godoc.org/github.com/monzo/typhon?status.svg)](https://godoc.org/github.com/monzo/typhon)

Typhon is a wrapper around Go's [`net/http`] library that we use at Monzo to build RPC servers and clients in [our microservices platform][platform blog post].

It provides a number of conveniences and tries to promote safety wherever possible. Here's a short list of interesting features in Typhon:

* **No need to close `body.Close()` in clients**  
  Forgetting to `body.Close()` in a client when the body has been dealt with is a common source of resource leaks in Go programs in our experience. Typhon ensures that – unless you're doing something really weird with the body – it will be closed automatically.

* **Middleware "filters"**  
  Filters are decorators around `Service`s; in Typhon servers and clients share common functionality by composing it functionally.

* **Body encoding and decoding**  
  Marshalling and unmarshalling request bodies to structs is such a common operation that our `Request` and `Response` objects support them directly. If the operations fail, the errors are propagated automatically since that's nearly always what a server will want.

* **Propagation of cancellation**  
  When a server has done handling a request, the request's context is automatically cancelled, and these cancellations are propagated through the distributed call stack. This lets downstream servers conserve work producing responses that are no longer needed.

* **Error propagation**  
  Responses have an inbuilt `Error` attribute, and serialisation/deserialisation of these errors into HTTP errors is taken care of automatically. We recommend using this in conjunction with [`monzo/terrors`].

* **Full HTTP/1.1 and HTTP/2.0 support**  
  Applications implemented using Typhon can communicate over HTTP/1.1 or HTTP/2.0. Typhon has support for full duplex communication under HTTP/2.0, and [`h2c`] (HTTP/2.0 over TCP, ie. without TLS) is also supported if required.

[`net/http`]: https://golang.org/pkg/net/http/
[platform blog post]: https://monzo.com/blog/2016/09/19/building-a-modern-bank-backend/
[`monzo/terrors`]: http://github.com/monzo/terrors
[`h2c`]: https://httpwg.org/specs/rfc7540.html#discover-http
