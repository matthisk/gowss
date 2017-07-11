package websocket

import (
	"io"
)

type fragmentReader struct {
	R io.Reader
	H FrameHandler
}

func (r *fragmentReader) Read(b []byte) (n int, err error) {
	n, err = r.R.Read(b)

	if err == io.EOF || n < len(b) {
		opcode, reader, err := r.H.NextReader()

		// If this was the last frame
		if opcode != ContinuationFrame {
			return n, err
		}

		// Otherwise we continue the read on the next fragment
		r.R = reader

		return r.Read(b[n:])
	}

	return n, err
}
