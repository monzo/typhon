package typhon

import (
	"bytes"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"

	"github.com/monzo/typhon/prototest"
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

func BenchmarkResponseProtobufDecode(b *testing.B) {
	b.ReportAllocs()

	g := &prototest.Greeting{Message: "Hello world!", Priority: 1}
	out, _ := proto.Marshal(g)
	rsp := NewResponse(NewRequest(nil, "GET", "/", nil))
	rsp.Header.Set("Content-Type", "application/protobuf")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rsp.Body = ioutil.NopCloser(bytes.NewReader(out))
		g := &prototest.Greeting{}
		rsp.Decode(g)
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

func BenchmarkRepeatedProtobufDecode(b *testing.B) {
	b.ReportAllocs()

	g := &prototest.Greeting{Message: "Hello world!", Priority: 1}
	out, _ := proto.Marshal(g)
	rsp := NewResponse(NewRequest(nil, "GET", "/", nil))
	rsp.Header.Set("Content-Type", "application/protobuf")
	rsp.Body = &rc{*strings.NewReader(string(out)), 0}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gout := &prototest.Greeting{}
		rsp.Decode(gout)
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
