package slog

import (
	"log"
)

// StdlibLogger is a very simple logger which forwards events to Go's standard library logger
type StdlibLogger struct{}

func (s StdlibLogger) Log(evs ...Event) {
	for _, e := range evs {
		log.Print(e.String())
	}
}

func (s StdlibLogger) Flush() error {
	return nil
}
