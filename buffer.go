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

// Streamer returns a reader/writer/closer that can be used to stream responses. A simple use of this is:
//  func streamingService(req typhon.Request) typhon.Response {
//      body := typhon.Streamer()
//      go func() {
//          defer body.Close()
//          // do something to asynchronously produce output into body
//      }()
//      return req.Response(body)
//  }
//
// Note that a Streamer may not perform any internal buffering, so callers should take care not to depend on writes
// being non-blocking. If buffering is needed, Streamer can be wrapped in a bufio.Writer.
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
		// Some underlying reader implementations may not return io.EOF when they have been closed.
		// Returning EOF on this "final successful read" prevents consumers from erroring.
		if err == nil {
			err = io.EOF
		}
	}
	return n, err
}
