package slog

// A Logger is a way of outputting events.
type Logger interface {
	Log(evs ...Event)
	Flush() error
}

// A MultiLogger sends invocations to multiple Loggers.
type MultiLogger []Logger

func (ls MultiLogger) Log(evs ...Event) {
	for _, l := range ls {
		l.Log(evs...)
	}
}

func (ls MultiLogger) Flush() error {
	for _, l := range ls {
		if err := l.Flush(); err != nil {
			return err
		}
	}
	return nil
}
