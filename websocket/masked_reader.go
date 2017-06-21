package websocket

import (
	"io"
)

// MaskedReader can read from an io.Reader and unmask the bytes in
// this reader using MaskedReader.mask
// See rfc6455#section-5.1
type MaskedReader struct {
	rd     io.Reader
	mask   [4]byte
	offset int
}

// NewMaskedReader returns a masked reader with mask used for unmasking any read bytes
func NewMaskedReader(reader io.Reader, mask [4]byte) *MaskedReader {
	return &MaskedReader{reader, mask, 0}
}

func unmask(offset int, mask [4]byte, bytes []byte) {
	for i, b := range bytes {
		m := mask[(offset+i)%4]
		bytes[i] = b ^ m
	}
}

func (r *MaskedReader) Read(b []byte) (n int, err error) {
	n, err = r.rd.Read(b)

	unmask(r.offset, r.mask, b)

	r.offset += n

	return n, err
}
