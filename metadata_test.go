package typhon

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetadataRoundtrip(t *testing.T) {
	meta := NewMetadata(map[string]string{
		"meta": "data",
	})
	ctx := context.Background()

	withMeta := AppendMetadataToContext(ctx, meta)
	out := MetadataFromContext(withMeta)

	assert.Equal(t, meta, out)
}

func TestMetadataNotSet(t *testing.T) {
	meta := NewMetadata(map[string]string{})
	out := MetadataFromContext(context.Background())

	assert.Equal(t, meta, out)
}
