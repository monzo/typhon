# Typhon Uasge Guide 🐲

## Installation
```bash
go get github.com/monzo/typhon
```

### Define Routes

```go
router := typhon.Router{}

router.GET("/ping", PingHandlerFun)

router.POST("/foo", FooHandlerFun)
```

### Define Server & Add Middlewares
```go
svc := router.Serve().
	Filter(typhon.ErrorFilter).
	Filter(typhon.H2cFilter)
```

### Run HTTP Server
```go
srv, err := typhon.Listen(
  svc,
  "localhost:8000",
  typhon.WithTimeout(typhon.TimeoutOptions{Read: time.Second * 10}))

log.Printf("👋 Listening on %v", srv.Listener().Addr())
```

### Gracefully Stop Server
```go
done := make(chan os.Signal, 1)
signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
<-done
log.Printf("☠️ Shutting down")

c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
srv.Stop(c)
```

### Define Request Handler Function
```go
func demo(req typhon.Request) typhon.Response {
    appHttp := typhon.HttpFacade{Request: req}
    return appHttp.ResponseWithView(200, "./demo.html", nil)
}
```

## Description
Typhon is a Go HTTP framework by Monzo that simplifies the process of creating robust and scalable HTTP services. This README provides instructions for installation, defining routes, setting up a server with middleware, running the server, and gracefully shutting it down. Additionally, it includes an example of defining a request handler function.