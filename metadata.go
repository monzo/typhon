package typhon

import "context"

type metadataKey struct{}

// Metadata provides a transport agnostic way to pass metadata with Typhon.
// It aligns to the interface of Go's default HTTP header type for convenience.
type Metadata map[string][]string

// NewMetadata creates a metadata struct from a map of strings.
func NewMetadata(data map[string]string) Metadata {
	meta := make(Metadata, len(data))
	for k, v := range data {
		meta[k] = []string{v}
	}
	return meta
}

// AppendMetadataToContext sets the metadata on the context.
func AppendMetadataToContext(ctx context.Context, md Metadata) context.Context {
	return context.WithValue(ctx, metadataKey{}, md)
}

// MetadataFromContext retrieves the metadata from the context.
func MetadataFromContext(ctx context.Context) Metadata {
	meta, ok := ctx.Value(metadataKey{}).(Metadata)
	if !ok {
		return Metadata{}
	}
	return meta
}
