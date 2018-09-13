package slog

import (
	"bytes"

	"github.com/cihub/seelog"
)

var severityToSeelog = map[Severity]func(...interface{}){
	// Really don't get this. Critical, Error, Warn return errors, whereas Info, Debug, Trace do not.
	CriticalSeverity: func(v ...interface{}) { seelog.Critical(v...) },
	ErrorSeverity:    func(v ...interface{}) { seelog.Error(v...) },
	WarnSeverity:     func(v ...interface{}) { seelog.Warn(v...) },
	InfoSeverity:     seelog.Info,
	DebugSeverity:    seelog.Debug,
	TraceSeverity:    seelog.Trace}

func SeelogLogger() Logger {
	return seelogger{}
}

type seelogger struct{}

func (_ seelogger) Log(evs ...Event) {
	for _, ev := range evs {
		impl, ok := severityToSeelog[ev.Severity]
		if !ok {
			impl = seelog.Debug
		}
		buf := new(bytes.Buffer)
		buf.WriteString(ev.Message)
		if len(ev.Metadata) > 0 {
			buf.WriteString(" {")
			i := 0
			for k, v := range ev.Metadata {
				if i != 0 {
					buf.WriteString(", ")
				}
				buf.WriteString(k)
				buf.WriteRune('=')
				buf.WriteString(v)
				i++
			}
			buf.WriteRune('}')
		}
		impl(buf.String())
	}
}

func (_ seelogger) Flush() error {
	seelog.Flush()
	return nil
}
