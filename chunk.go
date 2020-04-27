package typhon

import (
	"io"
	"net/http"
)

// Best streaming performance (especially when passing through data from pull based sources,
// e.g. a service which is proxying read data from an SFTP server on the ocean) is achieved
// by using src.(io.WriterTo).WriteTo(dst). By doing this, we allow the source to take
// responsibility for extracting maximum parallelism from whatever underlying protocol is in
// use. Especially with these sorts of sources, the fallback src.Read()/dst.Write loop can
// produce very pessimistic results.
//
// However, we also wish to ensure that if the source is slow at producing output, we do not
// allow said data to sit in the destination's internal buffer forever. To avoid this, we
// implement our own writer here which can be used as the target of WriterTo, but will also
// regularly flush the underlying writer. Our logic is:
// * We will accept writes from the source into our internal buffer
// * In parallel, we will attempt to empty that internal buffer into the backing writer
// * Each time our internal buffer empties (i.e. we are faster than our source), we will
//   flush the underlying stream to avoid keeping the reader waiting
//
// We use an internal ring buffer to handle this
type flusherWriter struct {
	dst      io.Writer
	err      error
	buf      []byte
	writePos int
	writeCap int

	// The destination side returns slices it has written out to the
	// source side via this channel. If the destination side encounters
	// an error, it places it in err and closes this channel
	empty chan writerChunk

	// The source side places slices contianing data in this channel. When
	// the writer is closed, this channel will also be closed
	full chan writerChunk
}

// A chunk being passed between the two halves. `The cases are
// * buf == nil -> Sender explicit flush
// * buf != nil -> Data to write
//     * advance == 0 -> Source buffer passed directly
//     * advance != 0 -> Slice of our internal buffer passed
type writerChunk struct {
	buf     []byte
	advance int
}

// Check interface completeness
var _ io.WriteCloser = &flusherWriter{}
var _ io.ReaderFrom = &flusherWriter{}
var _ http.Flusher = &flusherWriter{}

func newFlusherWriter(dst io.Writer, buf []byte) *flusherWriter {
	// If no buffer provided, allocate a default sized one
	if len(buf) == 0 {
		buf = make([]byte, 32*1024)
	}

	w := &flusherWriter{
		dst:      dst,
		err:      nil,
		buf:      buf,
		writePos: 0,
		writeCap: len(buf),
		// These sizes are entirely arbitrarily chosen
		empty: make(chan writerChunk, 4),
		full:  make(chan writerChunk, 4),
	}

	go w.run()
	return w
}

//
// Source side logic
//

// Drain any inbound empty buffers. We do this in
// part to determine if the destination has returned
// an error
func (f *flusherWriter) drainEmpties() error {
	for {
		select {
		case chunk, ok := <-f.empty:
			if !ok {
				return f.err
			}
			f.writeCap += chunk.advance

		default:
			return nil
		}
	}
}

// Send a chunk to the destination side. While attempting
// to send the chunk, we will drain any empties that are
// returned in order to both ensure we aren't blocking the
// destination side's progress, and also test for an error
// being returned
func (f *flusherWriter) sendChunk(c writerChunk) error {
	for {
		select {
		case chunk, ok := <-f.empty:
			if !ok {
				return f.err
			}
			f.writeCap += chunk.advance

		case f.full <- c:
			return nil
		}
	}
}

// Waits until a zero-sized empty is returned (i.e. our
// source buffer has been processed, and checks that it
// is our buffer
func (f *flusherWriter) waitTilProcessed(buf []byte) error {
	for chunk := range f.empty {
		if chunk.advance == 0 {
			// Sanity check - this should be impossible, as there
			// should be no case where its possible to have two such
			// writes outstandiing
			if len(chunk.buf) != len(buf) || &chunk.buf[0] != &buf[0] {
				panic("wrong buf returned by destination side")
			}

			return nil
		}
	}
	// The destintion side was closed - return their error
	return f.err
}

// Process emtpies until we have a viable target buffer
func (f *flusherWriter) getWriteBuffer() ([]byte, error) {
	// Optimistically drain any empty buffers
	if err := f.drainEmpties(); err != nil {
		return nil, err
	}

	// Wait until we have any space
	for f.writeCap == 0 {
		chunk, ok := <-f.empty
		if !ok {
			return nil, f.err
		}
		f.writeCap += chunk.advance
	}

	// If the entire buffer is free, then reset our position to zero
	// so that we can maximise the size of our contiguous reads/writes
	if f.writeCap == len(f.buf) {
		f.writePos = 0
	}

	// Grab as many bytes as we can contiguously from our write position
	end := f.writePos + f.writeCap
	if end > len(f.buf) {
		end = len(f.buf)
	}

	return f.buf[f.writePos:end], nil
}

// Indicates how much we've written to the last returned buffer and
// pushes to the destination size
func (f *flusherWriter) written(nb int) error {
	buf := f.buf[f.writePos : f.writePos+nb]
	f.writeCap -= nb
	f.writePos += nb
	switch {
	case f.writePos == len(f.buf):
		// Wrap
		f.writePos = 0
	case f.writePos > len(f.buf):
		// Should never happen and in facr we probably crashed above
		// while forming the slice anyway
		panic("wrote beyond length of our internal buffer")
	}

	// it's entirely possible that we might be trying to send an
	// n-byte buffer here, but while we wait to send that we gain
	// more capacity in our write channel. In such cases it would
	// be preferable for us to to increase the amount of data that
	// we include in this chunk.
	//
	// We can leave this as an optimisation for later though
	return f.sendChunk(writerChunk{
		buf:     buf,
		advance: nb,
	})
}

// satisfies io.Writer
func (f *flusherWriter) Write(buf []byte) (int, error) {
	if f.full == nil {
		return 0, io.ErrClosedPipe
	}

	// Pass large buffers directly to the draining side
	if len(buf) > len(f.buf) {
		if err := f.sendChunk(writerChunk{buf: buf}); err != nil {
			return 0, err
		}

		if err := f.waitTilProcessed(buf); err != nil {
			return 0, err
		}
		return len(buf), nil
	}

	// Handle shorter writes by copying through our internal buffer
	totalWritten := 0
	for len(buf) > 0 {
		destBuf, err := f.getWriteBuffer()
		if err != nil {
			return totalWritten, err
		}

		nb := copy(destBuf, buf)
		if err := f.written(nb); err != nil {
			return totalWritten, err
		}
		totalWritten += nb
	}
	return totalWritten, nil
}

// satisfies http.Flusher
func (f *flusherWriter) Flush() {
	// Check if we're closed
	if f.full == nil {
		return
	}
	// Since we don't have an error return here, we have to discard
	// any returned error. That's OK though: a future Write or Close
	// will find it
	_ = f.sendChunk(writerChunk{buf: nil, advance: 0})
}

// satisifes io.ReaderFrom
func (f *flusherWriter) ReadFrom(r io.Reader) (n int64, err error) {
	// Check if we are closed
	if f.full == nil {
		return 0, io.ErrClosedPipe
	}

	for {
		var destBuf []byte
		var nb int

		destBuf, err = f.getWriteBuffer()
		if err != nil {
			break
		}

		nb, err = r.Read(destBuf)
		n += int64(nb)
		if nb == 0 || err != nil {
			break
		}

		err = f.written(nb)
		if err != nil {
			break
		}
	}

	if err == io.EOF {
		err = nil
	}

	return n, err
}

// satisfies io.WriteCloser
// Closure happens from the source side, and then waits for
// the destination side to drain
func (f *flusherWriter) Close() error {
	if f.full != nil {
		close(f.full)
		for _ = range f.empty {
			// Pump the returns channel until closure
		}

		// For inexplicable reasons, reading from a nil channel blocks
		// forever. Wait until run() has finished before clearing this
		// (our closure signal)
		f.full = nil
	}
	// Return any stored error
	return f.err
}

//
// Destination side logic
//

func (f *flusherWriter) flushDestination() {
	if flusher, ok := f.dst.(http.Flusher); ok {
		flusher.Flush()
	}
}

// writeChunk handles writing a chunk to our destination stream
func (f *flusherWriter) writeChunk(chunk writerChunk) (flushed bool, err error) {
	// If there's no buf, this is a flush
	if chunk.buf == nil {
		f.flushDestination()
		return true, nil
	}

	// Otherwise, this is a write
	nw, err := f.dst.Write(chunk.buf)
	if err != nil {
		return false, err
	} else if nw != len(chunk.buf) {
		return false, io.ErrShortWrite
	}

	// Return the chunk to the source side
	f.empty <- chunk
	return false, nil
}

// This is the "destination side" of a flusherWriter, responsible for
// copying read data to the backing stream.
func (f *flusherWriter) run() {
	var flushed bool
	var err error
outer:
	for chunk := range f.full {
		flushed, err = f.writeChunk(chunk)
		if err != nil {
			break outer
		}

		// As long as the source can provide data faster than our destination
		// is able to accept it, there's no need for us to explicitly flush:
		// the flow of data is guaranteeing it is eventually evicted (unless
		// an underlying buffer is infinite in size, but we should not worry
		// about such absurd situations)
	inner:
		for {
			select {
			case chunk, ok := <-f.full:
				if !ok {
					break inner
				}

				flushed, err = f.writeChunk(chunk)
				if err != nil {
					break outer
				}
			default:
				break inner
			}
		}

		// We now need to block for more data from the source
		// If we haven't already flushed, we should do so
		if !flushed {
			f.flushDestination()
		}
	}

	// Either the data channel was just closed, or we encountered an error
	// Handle both
	if err != nil {
		// We hit an error. Shut us down
		f.err = err
	} else if !flushed {
		// The other side initiated a close - must have ran out of data
		// If we haven't flushed the last data, do so, then close the empty
		// buffer reutrn channel and stop
		f.flushDestination()
	}
	close(f.empty)
}

func copyChunked(dst io.Writer, src io.Reader, buf []byte) (written int64, err error) {
	// If the stream doesn't expose http.Flusher, there's no need for us to do anything
	// special. We just delegate to the IO library and let it optimise it as it prefers
	if _, isFlusher := dst.(http.Flusher); !isFlusher {
		// TODO: Depending upon circumstances flusherWriter may be faster, as it can
		// exploit greater read/write parallelism. Should we just always prefer it?
		// For now, preserving existing behaviour
		return io.CopyBuffer(dst, src, buf)
	}

	fw := newFlusherWriter(dst, buf)

	// Defer closing our writer. This ensures we don't leak the internal goroutine
	// even if we panic inside WriteTo/ReadFrom
	defer func() {
		// Close and flush our writer. If both Close and WriteTo/ReadFrom return an error,
		// we prefer the earlier error
		closeErr := fw.Close()
		if err == nil && closeErr != io.EOF {
			err = closeErr
		}
	}()
	// Mysteriously, Go's http2 implementation doesn't write response headers until there is at least one byte of the
	// body available. Code comments indicate that is deliberate, but it isn't desirable for us. Calling Flush()
	// forces headers to be sent.
	fw.Flush()

	// If the source has WriterTo, prefer that. Otherwise use our ReadFrom.
	// These are the same preferences as expressed by io.Copy
	if writerTo, ok := src.(io.WriterTo); ok {
		written, err = writerTo.WriteTo(fw)
	} else {
		written, err = fw.ReadFrom(src)
	}

	// As per io.Copy, suppress EOF errors:
	// A successful Copy returns err == nil, not err == EOF. Because Copy is defined to read from src until EOF,
	// it does not treat an EOF from Read as an error to be reported.
	if err == io.EOF {
		err = nil
	}

	return
}
