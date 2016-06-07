package typhon

import (
	"bytes"
)

type bufCloser struct {
	bytes.Buffer
}

func (b *bufCloser) Close() error {
	return nil // No-op
}
