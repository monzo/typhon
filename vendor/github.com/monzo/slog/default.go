package slog

import (
	"context"
	"sync"
)

var (
	defaultLogger  Logger = StdlibLogger{}
	defaultLoggerM sync.RWMutex
)

func DefaultLogger() Logger {
	defaultLoggerM.RLock()
	defer defaultLoggerM.RUnlock()
	return defaultLogger
}

func SetDefaultLogger(l Logger) {
	defaultLoggerM.Lock()
	defer defaultLoggerM.Unlock()
	defaultLogger = l
}

// Log sends the given Events via the default Logger
func Log(evs ...Event) {
	if l := DefaultLogger(); l != nil {
		l.Log(evs...)
	}
}

// Critical constructs a logging event with critical severity. If the
// default Logger implements the LeveledLogger interface, we forward the
// requests via the Critical interface function. If not, the event is sent
// via the default Logger
func Critical(ctx context.Context, msg string, params ...interface{}) {
	if l := DefaultLogger(); l != nil {
		if ll, ok := l.(LeveledLogger); ok {
			ll.Critical(ctx, msg, params...)
		} else {
			l.Log(Eventf(CriticalSeverity, ctx, msg, params...))
		}
	}
}

// Error constructs a logging event with error severity. If the
// default Logger implements the LeveledLogger interface, we forward the
// requests via the Error interface function. If not, the event is sent
// via the default Logger
func Error(ctx context.Context, msg string, params ...interface{}) {
	if l := DefaultLogger(); l != nil {
		if ll, ok := l.(LeveledLogger); ok {
			ll.Error(ctx, msg, params...)
		} else {
			l.Log(Eventf(ErrorSeverity, ctx, msg, params...))
		}
	}
}

// Warn constructs a logging event with warn severity. If the
// DefaultLogger implements the LeveledLogger interface, we forward the
// requests via the Warn interface function. If not, the event is sent
// via the default Logger
func Warn(ctx context.Context, msg string, params ...interface{}) {
	if l := DefaultLogger(); l != nil {
		if ll, ok := l.(LeveledLogger); ok {
			ll.Warn(ctx, msg, params...)
		} else {
			l.Log(Eventf(WarnSeverity, ctx, msg, params...))
		}
	}
}

// Info constructs a logging event with info severity. If the
// default Logger implements the LeveledLogger interface, we forward the
// requests via the Info interface function. If not, the event is sent
// via the default Logger
func Info(ctx context.Context, msg string, params ...interface{}) {
	if l := DefaultLogger(); l != nil {
		if ll, ok := l.(LeveledLogger); ok {
			ll.Info(ctx, msg, params...)
		} else {
			l.Log(Eventf(InfoSeverity, ctx, msg, params...))
		}
	}
}

// Debug constructs a logging event with debug severity. If the
// default Logger implements the LeveledLogger interface, we forward the
// requests via the Debug interface function. If not, the event is sent
// via the default Logger
func Debug(ctx context.Context, msg string, params ...interface{}) {
	if l := DefaultLogger(); l != nil {
		if ll, ok := l.(LeveledLogger); ok {
			ll.Debug(ctx, msg, params...)
		} else {
			l.Log(Eventf(DebugSeverity, ctx, msg, params...))
		}
	}
}

// Trace constructs a logging event with trace severity. If the
// default Logger implements the LeveledLogger interface, we forward the
// requests via the Trace interface function. If not, the event is sent
// via the default Logger
func Trace(ctx context.Context, msg string, params ...interface{}) {
	if l := DefaultLogger(); l != nil {
		if ll, ok := l.(LeveledLogger); ok {
			ll.Trace(ctx, msg, params...)
		} else {
			l.Log(Eventf(TraceSeverity, ctx, msg, params...))
		}
	}
}
