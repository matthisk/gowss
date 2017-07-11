package websocket

import (
	"io"
)

// PayloadReader is an interface resembling a payload reader
type PayloadReader interface {
	io.Reader

	PayloadType() byte
	Len() int64
	Remaining() int64
}

// payloadReader read the remaining bytes from a websocket frame
type payloadReader struct {
	reader        io.Reader
	header        FrameHeader
	readRemaining int64
}

// NewPayloadReader returns a payload reader struct
func NewPayloadReader(reader io.Reader, header FrameHeader) PayloadReader {
	if header.mask {
		reader = NewMaskedReader(reader, header.maskBytes)
	}

	return &payloadReader{reader, header, header.payloadLength}
}

// PayloadType returns the opcode of the current frame
func (r payloadReader) PayloadType() byte {
	return r.header.opcode
}

// Len returns the total amount of bytes in this frame's payload
func (r payloadReader) Len() int64 {
	return r.header.payloadLength
}

// Remaining returns the remaining bytes in this frame's payload
func (r payloadReader) Remaining() int64 {
	return r.readRemaining
}

func (r *payloadReader) Read(b []byte) (n int, err error) {
	if r.readRemaining == 0 {
		return 0, io.EOF
	}

	if int64(len(b)) > r.readRemaining {
		b = b[:r.readRemaining]

		n, err := r.reader.Read(b)

		r.readRemaining -= int64(n)

		return n, err
	}

	n, err = r.reader.Read(b)

	r.readRemaining -= int64(n)

	return n, err
}
