package slog

import (
	"bytes"
)

// EventSet is a time-sortable collection of logging events.
type EventSet []Event

func (es EventSet) Len() int {
	return len(es)
}

func (es EventSet) Swap(i, j int) {
	es[i], es[j] = es[j], es[i]
}

func (es EventSet) Less(i, j int) bool {
	return es[i].Timestamp.Before(es[j].Timestamp)
}

func (es EventSet) String() string {
	buf := new(bytes.Buffer)
	for i := 0; i < len(es); i++ {
		e := es[i]
		if i > 0 {
			buf.WriteRune('\n')
		}
		buf.WriteString(e.String())
	}
	return buf.String()
}
