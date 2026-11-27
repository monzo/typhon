package typhon

import (
	"mime"
	"net/http"
	"strings"
)

var protobufMediaTypes = map[string]struct{}{
	"application/octet-stream":      {},
	"application/x-google-protobuf": {},
	"application/protobuf":          {},
	"application/x-protobuf":        {},
}

func canonicalMediaType(value string) string {
	if value == "" {
		return ""
	}

	mediaType, _, err := mime.ParseMediaType(value)
	if err == nil {
		return strings.ToLower(mediaType)
	}

	return strings.ToLower(strings.TrimSpace(strings.SplitN(value, ";", 2)[0]))
}

func isProtobufMediaType(value string) bool {
	_, ok := protobufMediaTypes[canonicalMediaType(value)]
	return ok
}

func acceptsProtobuf(header http.Header) bool {
	if header == nil {
		return false
	}

	for _, acceptValue := range header.Values("Accept") {
		for _, part := range strings.Split(acceptValue, ",") {
			mediaType, params, err := mime.ParseMediaType(strings.TrimSpace(part))
			if err != nil {
				mediaType = canonicalMediaType(part)
				params = nil
			}

			if params != nil && params["q"] == "0" {
				continue
			}

			mediaType = strings.ToLower(mediaType)
			if mediaType == "*/*" || mediaType == "application/*" || isProtobufMediaType(mediaType) {
				return true
			}
		}
	}

	return false
}
