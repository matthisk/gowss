package websocket

import (
	"io"
)

// PayloadReader read the remaining bytes from a websocket frame
type PayloadReader struct {
	reader        io.Reader
	header        FrameHeader
	readRemaining int
}

// NewPayloadReader returns a payload reader struct
func NewPayloadReader(reader io.Reader, header FrameHeader) PayloadReader {
	if header.mask {
		reader = NewMaskedReader(reader, header.maskBytes)
	}

	return PayloadReader{reader, header, header.payloadLength}
}

func (r *PayloadReader) Read(b []byte) (n int, err error) {
	if len(b) > r.readRemaining {
		b = b[:r.readRemaining]

		n, err := r.reader.Read(b)

		r.readRemaining -= n

		return n, err
	}

	return 0, io.EOF
}
