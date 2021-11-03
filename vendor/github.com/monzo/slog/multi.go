package slog

// A MultiLogger sends invocations to multiple Loggers.
type MultiLogger []Logger

// Log the event to each sub-logger.
func (ls MultiLogger) Log(evs ...Event) {
	for _, l := range ls {
		l.Log(evs...)
	}
}

// Flush all sub-loggers.
func (ls MultiLogger) Flush() error {
	for _, l := range ls {
		if err := l.Flush(); err != nil {
			return err
		}
	}
	return nil
}
