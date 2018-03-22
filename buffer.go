package typhon

import (
	"bytes"
	"io"
	"sync"
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

// countingWriter is a writer which proxies writes to an underlying io.Writer, keeping track of how many bytes have
// been written in total
type countingWriter struct {
	n int
	io.Writer
}

func (w *countingWriter) Write(p []byte) (int, error) {
	n, err := w.Writer.Write(p)
	w.n += n
	return n, err
}

// doneReader is a wrapper around a ReadCloser which provides notification when the stream has been fully consumed
// (ie. when EOF is reached, when the reader is explicitly closed, or if the size of the underlying reader is known,
// when it has been fully read [even if EOF is not reached.])
type doneReader struct {
	closed     chan struct{}
	closedOnce sync.Once
	length     int64 // length of the underlying reader in bytes, if known. ≤0 indicates unknown
	read       int64 // number of bytes read
	io.ReadCloser
}

func newDoneReader(r io.ReadCloser, length int64) *doneReader {
	return &doneReader{
		closed:     make(chan struct{}),
		length:     length,
		ReadCloser: r}
}

func (r *doneReader) Close() error {
	err := r.ReadCloser.Close()
	r.closedOnce.Do(func() { close(r.closed) })
	return err
}

func (r *doneReader) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	r.read += int64(n)
	// If we got an error reading, or the reader's length is known and is now exhausted, close
	// the underlying reader
	if err != nil || (r.length > 0 && r.read >= r.length) {
		r.Close()
	}
	return n, err
}
