package slog

import "context"

// A Logger is a way of outputting events.
type Logger interface {
	Log(evs ...Event)
	Flush() error
}

// LeveledLogger is a logger which logs at different levels.
type LeveledLogger interface {
	Critical(ctx context.Context, msg string, params ...interface{})
	Error(ctx context.Context, msg string, params ...interface{})
	Warn(ctx context.Context, msg string, params ...interface{})
	Info(ctx context.Context, msg string, params ...interface{})
	Debug(ctx context.Context, msg string, params ...interface{})
	Trace(ctx context.Context, msg string, params ...interface{})
}

// SeverityLogger is a logger which can log at different severity levels.
type SeverityLogger struct {
	Logger
}

// NewSeverityLogger creates a SeverityLogger which wraps the default logger.
func NewSeverityLogger() SeverityLogger {
	return SeverityLogger{
		Logger: DefaultLogger(),
	}
}

// Critical writes a Critical event to the logger.
func (s SeverityLogger) Critical(ctx context.Context, msg string, params ...interface{}) {
	s.Log(Eventf(CriticalSeverity, ctx, msg, params...))
}

// Error writes a Error event to the logger.
func (s SeverityLogger) Error(ctx context.Context, msg string, params ...interface{}) {
	s.Log(Eventf(ErrorSeverity, ctx, msg, params...))
}

// Warn writes a Warn event to the logger.
func (s SeverityLogger) Warn(ctx context.Context, msg string, params ...interface{}) {
	s.Log(Eventf(WarnSeverity, ctx, msg, params...))
}

// Info writes a Info event to the logger.
func (s SeverityLogger) Info(ctx context.Context, msg string, params ...interface{}) {
	s.Log(Eventf(InfoSeverity, ctx, msg, params...))
}

// Debug writes a Debug event to the logger.
func (s SeverityLogger) Debug(ctx context.Context, msg string, params ...interface{}) {
	s.Log(Eventf(DebugSeverity, ctx, msg, params...))
}

// Trace writes a Trace event to the logger.
func (s SeverityLogger) Trace(ctx context.Context, msg string, params ...interface{}) {
	s.Log(Eventf(TraceSeverity, ctx, msg, params...))
}
