package typhon

import (
	"syscall"

	"github.com/monzo/slog"
)

func copyErrnoSeverity(err syscall.Errno) slog.Severity {
	switch err {
	case syscall.EPIPE: // the client has cancelled the request
		return slog.DebugSeverity
	default:
		return slog.WarnSeverity
	}
}
