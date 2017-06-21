package websocket

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestReader(t *testing.T) {
	rd := strings.NewReader("Hello World!")
	mrd := NewMaskedReader(rd, [4]byte{0x0, 0x0, 0x0, 0x0})

	brd := bufio.NewReader(mrd)

	line, _, err := brd.ReadLine()

	if err != nil {
		t.Error(err)
	}

	if string(line) != "Hello World!" {
		t.Error("Expected `line` to equal `Hello World!`")
	}
}

func TestReaderMasked(t *testing.T) {
	input := []byte{0xa, 0xb, 0xc, 0xd, 0xe, 0xf, 0x1, 0x2}
	mask := [4]byte{0xf, 0x0, 0xf, 0x0}

	rd := bytes.NewReader(input)
	mrd := NewMaskedReader(rd, mask)

	bytes := make([]byte, 8)
	_, err := mrd.Read(bytes)

	if err != nil {
		t.Error(err)
	}

	for i, b := range bytes {
		m := mask[i%4]
		e := input[i] ^ m

		if e != b {
			t.Error("Expected e to equal b")
		}
	}
}
