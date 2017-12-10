package typhon

import (
	"bytes"
	"io"
)

type bufCloser struct {
	bytes.Buffer
}

func (b *bufCloser) Close() error {
	return nil // No-op
}

type streamer struct {
	pipeR *io.PipeReader
	pipeW *io.PipeWriter
}

// Streamer returns a reader/writer/closer that can be used to stream service responses. It does not necessarily
// perform internal buffering, so users should take care not to depend on such behaviour.
func Streamer() io.ReadWriteCloser {
	pipeR, pipeW := io.Pipe()
	return &streamer{
		pipeR: pipeR,
		pipeW: pipeW}
}

func (s *streamer) Read(p []byte) (int, error) {
	return s.pipeR.Read(p)
}

func (s *streamer) Write(p []byte) (int, error) {
	return s.pipeW.Write(p)
}

func (s *streamer) Close() error {
	return s.pipeW.Close()
}
