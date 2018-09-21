// +build !linux,!darwin

package typhon

import (
	"syscall"

	"github.com/monzo/slog"
)

func copyErrnoSeverity(err syscall.Errno) slog.Severity {
	return slog.WarnSeverity
}
