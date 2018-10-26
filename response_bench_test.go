package typhon

import (
	"strings"
	"testing"
)

func BenchmarkResponseDecode(b *testing.B) {
	b.ReportAllocs()
	rsp := NewResponse(NewRequest(nil, "GET", "/", nil))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rsp.Body = &rc{*strings.NewReader(`{"foo":"bar"}` + "\n"), 0}
		v := map[string]string{}
		rsp.Decode(&v)
	}
}

func BenchmarkRepeatedResponseDecode(b *testing.B) {
	b.ReportAllocs()
	rsp := NewResponse(NewRequest(nil, "GET", "/", nil))
	rsp.Body = &rc{*strings.NewReader(`{"foo":"bar"}` + "\n"), 0}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := map[string]string{}
		rsp.Decode(&v)
	}
}

func BenchmarkRepeatedResponseBodyBytes(b *testing.B) {
	b.ReportAllocs()
	rsp := NewResponse(NewRequest(nil, "GET", "/", nil))
	rsp.Body = &rc{*strings.NewReader(`{"foo":"bar"}` + "\n"), 0}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rsp.BodyBytes(false)
	}
}
