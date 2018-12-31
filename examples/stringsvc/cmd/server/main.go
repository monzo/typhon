package main

import (
	"context"
	"github.com/monzo/typhon"
	"github.com/monzo/typhon/examples/stringsvc/internal/app/stringsvc/service"
	"github.com/monzo/typhon/examples/stringsvc/internal/app/stringsvc/transport"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	svc := service.New()
	typhonHandler := transport.NewHTTPTransport(svc).Filter(typhon.ErrorFilter).Filter(typhon.H2cFilter)

	srv, err := typhon.Listen(typhonHandler, ":8085")
	if err != nil {
		panic(err)
	}
	log.Printf("Listening on %v", srv.Listener().Addr())

	done := make(chan os.Signal)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	<-done

	log.Printf("Shutting down")
	c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Stop(c)
}
