package slog

import (
	"context"
	"fmt"
	"time"

	uuid "github.com/nu7hatch/gouuid"
)

type Severity int

const (
	ErrorMetadataKey          = "error"
	TimeFormat                = "2006-01-02 15:04:05-0700 (MST)"
	TraceSeverity    Severity = 1
	DebugSeverity    Severity = 2
	InfoSeverity     Severity = 3
	WarnSeverity     Severity = 4
	ErrorSeverity    Severity = 5
	CriticalSeverity Severity = 6
)

func (s Severity) String() string {
	switch s {
	case CriticalSeverity:
		return "CRITICAL"
	case ErrorSeverity:
		return "ERROR"
	case WarnSeverity:
		return "WARN"
	case InfoSeverity:
		return "INFO"
	case DebugSeverity:
		return "DEBUG"
	default:
		return "TRACE"
	}
}

type logMetadataProvider interface {
	LogMetadata() map[string]string
}

// An Event is a discrete logging event
type Event struct {
	Context         context.Context `json:"-"`
	Id              string          `json:"id"`
	Timestamp       time.Time       `json:"timestamp"`
	Severity        Severity        `json:"severity"`
	Message         string          `json:"message"`
	OriginalMessage string          `json:"-"`
	// Metadata are structured key-value pairs which describe the event.
	Metadata map[string]interface{} `json:"meta,omitempty"`
	// Labels, like Metadata, are key-value pairs which describe the event. Unlike Metadata, these are intended to be
	// indexed.
	Labels map[string]string `json:"labels,omitempty"`
	Error  interface{}       `json:"error,omitempty"`
}

func (e Event) String() string {
	errorMessage := ""
	if e.Error != nil {
		if err, ok := e.Error.(error); ok {
			errorMessage = err.Error()
		}
	}

	return fmt.Sprintf("[%s] %s %s (error=%v metadata=%v labels=%v id=%s)", e.Timestamp.Format(TimeFormat),
		e.Severity.String(), e.Message, errorMessage, e.Metadata, e.Labels, e.Id)
}

// Eventf constructs an event from the given message string and formatting operands. Optionally, event metadata
// (map[string]interface{}, or map[string]string) can be provided as a final argument.
func Eventf(sev Severity, ctx context.Context, msg string, params ...interface{}) Event {
	originalMessage := msg
	if ctx == nil {
		ctx = context.Background()
	}

	id, err := uuid.NewV4()
	if err != nil {
		return Event{}
	}

	metadata := map[string]interface{}(nil)
	var errParam error
	if len(params) > 0 {

		fmtOperands := countFmtOperands(msg)

		// If we have been provided with more params than we have formatting arguments, then we have
		// been given some metadata.
		extraParamCount := len(params) - fmtOperands

		// If we've got more fmtOperands than params, we have an invalid log statement.
		// In this case, we do our best to extract metadata from existing params, and
		// we also write as many as we can into the string.
		// For example, if you give: log("foo %s %s", err), we'll end up with "foo {err} %!s(MISSING)"
		// _and_ metadata extracted from the error.
		//
		// We do this so that we have the highest chance of actually capturing important details.
		// The alternative is erroring loudly, but as we see this in the wild for cases which are
		// rarely exercised and probably not covered in tests (e.g. error paths), I don't think
		// there's a better alternative.
		hasFormatOverflow := false
		if extraParamCount < 0 {
			hasFormatOverflow = true
			extraParamCount = len(params)
		}

		// Attempt to pull metadata and errors from any params.
		// This means that we'll still extract errors and metadata, even if it
		// is going to be interpolated into the message. This may result in some
		// duplication, but always gives us the most structured data possible.
		if len(params) > 0 {
			metadata = mergeMetadata(metadata, metadataFromParams(params))
			errParam = extractFirstErrorParam(params)
		}

		// If any of the provided params can be "upgraded" to a logMetadataProvider i.e.
		// they themselves have a LogMetadata method that returns a map[string]string
		// then we merge these params with the metadata.
		for _, param := range params {
			param, ok := param.(logMetadataProvider)
			if !ok {
				continue
			}
			metadata = mergeMetadata(metadata, stringMapToInterfaceMap(param.LogMetadata()))
		}

		if fmtOperands > 0 {
			endIndex := len(params) - extraParamCount
			if hasFormatOverflow {
				endIndex = len(params)
			}
			nonMetaParams := params[0:endIndex]
			msg = fmt.Sprintf(msg, nonMetaParams...)
		}
	}

	event := Event{
		Context:         ctx,
		Id:              id.String(),
		Timestamp:       time.Now().UTC(),
		Severity:        sev,
		Message:         msg,
		OriginalMessage: originalMessage,
		Metadata:        metadata,
		Error:           errParam,
	}

	return event
}

func extractFirstErrorParam(params []interface{}) error {
	for _, param := range params {
		err, ok := param.(error)
		if !ok {
			continue
		}
		return err
	}

	return nil
}

func metadataFromParams(params []interface{}) map[string]interface{} {
	result := map[string]interface{}(nil)
	for _, param := range params {
		// This is deprecated, but continue to support a map of strings.
		if metadataParam, ok := param.(map[string]string); ok {
			result = mergeMetadata(result, stringMapToInterfaceMap(metadataParam))
		}

		// Check for 'raw' metadata rather than strings.
		if metadataParam, ok := param.(map[string]interface{}); ok {
			result = mergeMetadata(result, metadataParam)
		}
	}
	return result
}

func stringMapToInterfaceMap(m map[string]string) map[string]interface{} {
	shim := make(map[string]interface{}, len(m))
	for k, v := range m {
		shim[k] = v
	}
	return shim
}

// mergeMetadata merges the metadata but preserves existing entries
func mergeMetadata(current, new map[string]interface{}) map[string]interface{} {
	if len(new) == 0 {
		return current
	}

	if current == nil {
		current = map[string]interface{}{}
	}

	for k, v := range new {
		if _, ok := current[k]; !ok {
			current[k] = v
		}
	}

	return current
}
