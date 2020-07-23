// totally stolen from https://github.com/stvp/rollbar/blob/master/stack.go
package stack

import (
	"fmt"
	"hash/crc32"
	"os"
	"runtime"
	"strings"
)

var (
	knownFilePathPatterns []string = []string{
		"github.com/",
		"code.google.com/",
		"bitbucket.org/",
		"launchpad.net/",
	}
)

type Frame struct {
	Filename string  `json:"filename"`
	Method   string  `json:"method"`
	Line     int     `json:"lineno"`
	PC       uintptr `json:"pc"`
}

type Stack []*Frame

func BuildStack(skip int) Stack {
	stack := make(Stack, 0)

	// Look up to a maximum depth of 100
	ret := make([]uintptr, 100)

	// Note that indexes must be one higher when passed to Callers()
	// than they would be when passed to Caller()
	// see https://golang.org/pkg/runtime/#Caller
	index := runtime.Callers(skip+1, ret)
	if index == 0 {
		// We have no frames to report, skip must be too high
		return stack
	}

	// This function takes a list of counters and gets function/file/line information
	cf := runtime.CallersFrames(ret[:index])

	for {
		frame, ok := cf.Next()
		stack = append(stack, &Frame{
			Filename: shortenFilePath(frame.File),
			Method:   functionName(frame.PC),
			Line:     frame.Line,
			PC:       frame.PC,
		})
		if !ok {
			// This was the last valid caller
			break
		}
	}
	return stack
}

// Create a fingerprint that uniquely identify a given message. We use the full
// callstack, including file names. That ensure that there are no false
// duplicates but also means that after changing the code (adding/removing
// lines), the fingerprints will change. It's a trade-off.
func (s Stack) Fingerprint() string {
	hash := crc32.NewIEEE()
	for _, frame := range s {
		fmt.Fprintf(hash, "%s%s%d", frame.Filename, frame.Method, frame.Line)
	}
	return fmt.Sprintf("%x", hash.Sum32())
}

// Remove un-needed information from the source file path. This makes them
// shorter in Rollbar UI as well as making them the same, regardless of the
// machine the code was compiled on.
//
// Examples:
//   /usr/local/go/src/pkg/runtime/proc.c -> pkg/runtime/proc.c
//   /home/foo/go/src/github.com/rollbar/rollbar.go -> github.com/rollbar/rollbar.go
func shortenFilePath(s string) string {
	idx := strings.Index(s, "/src/pkg/")
	if idx != -1 {
		return s[idx+5:]
	}
	for _, pattern := range knownFilePathPatterns {
		idx = strings.Index(s, pattern)
		if idx != -1 {
			return s[idx:]
		}
	}
	return s
}

func functionName(pc uintptr) string {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "???"
	}
	name := fn.Name()
	end := strings.LastIndex(name, string(os.PathSeparator))
	return name[end+1:]
}
