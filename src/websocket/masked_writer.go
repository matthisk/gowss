package websocket

import (
	"io"
)

// MaskedWriter write to a io.Writer using a 4 byte mask
type MaskedWriter struct {
	writer io.Writer
	mask   [4]byte
	offset int
}

// NewMaskedWriter returns a masked writer
func NewMaskedWriter(writer io.Writer, mask [4]byte) *MaskedWriter {
	return &MaskedWriter{writer, mask, 0}
}

func mask(offset int, mask [4]byte, bytes []byte) {
	for i, b := range bytes {
		m := mask[(offset+i)%4]
		bytes[i] = m ^ b
	}
}

// Writer mask input bytes and write to the underlying writer
func (wr *MaskedWriter) Write(b []byte) (n int, err error) {
	mask(wr.offset, wr.mask, b)

	n, err = wr.writer.Write(b)

	wr.offset += n

	return n, err
}
