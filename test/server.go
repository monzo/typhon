package test

import (
	"testing"
	"time"

	"github.com/b2aio/typhon/server"
)

// InitServer for testing
func InitServer(t *testing.T, name string) server.Server {
	// Initialize our Server
	server.Init(&server.Config{
		Name:        name,
		Description: "Example service",
	})

	go server.Run()

	select {
	case <-server.NotifyConnected():
	case <-time.After(10 * time.Second):
		t.Fatalf("Test Server couldn't connect to RabbitMQ")
	}

	return server.DefaultServer
}
