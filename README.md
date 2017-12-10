# Typhon 🐲

[![Build Status](https://travis-ci.org/monzo/typhon.svg?branch=master)](https://travis-ci.org/monzo/typhon)
[![GoDoc](https://godoc.org/github.com/monzo/typhon?status.svg)](https://godoc.org/github.com/monzo/typhon)

Typhon is a thin wrapper around `net/http` that we use at Monzo to build RPC servers and clients in our
microservices platform.

It provides a number of conveniences for doing things like injecting middleware "filters", encoding and decoding
responses, response streaming, propagating cancellation, passing errors. Its API is deliberately constrained but
intended to promote safety: for example, clients are freed from the worry of leaking resources if they fail call
`body.Close()`. By modelling servers, clients, and filters as straightforward functions, they are decoupled from the
underlying HTTP mechanisms, thus simplifying testing and promoting composition.
