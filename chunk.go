package typhon

import (
	"io"
	"net/http"
)

func copyChunked(dst io.Writer, src io.Reader, buf []byte) (written int64, err error) {
	flusher, flusherOk := dst.(http.Flusher)
	if !flusherOk {
		return io.Copy(dst, src)
	}

	// Mysteriously, Go's http2 implementation doesn't write response headers until there is at least one byte of the
	// body available. Code comments indicate that is deliberate, but it isn't desirable for us. Calling Flush()
	// forces headers to be sent.
	flusher.Flush()

	// This is taken and lightly adapted from the source of io.Copy
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			flusher.Flush()
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return
}
