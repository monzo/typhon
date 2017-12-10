package typhon

import (
	"io"
	"net/http"
)

func copyChunked(dst io.Writer, src io.Reader) (written int64, err error) {
	flusher, flusherOk := dst.(http.Flusher)
	if !flusherOk {
		return io.Copy(dst, src)
	}

	// This is taken and lightly adapted from the source of io.Copy
	buf := make([]byte, 32*1024)
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
			if flusherOk {
				flusher.Flush()
			}
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
