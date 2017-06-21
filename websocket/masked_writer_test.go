package websocket

import (
	"bufio"
	"bytes"
	"testing"
)

func TestWriter(t *testing.T) {
	wd := bytes.NewBufferString("")
	mwd := NewMaskedWriter(wd, [4]byte{0x0, 0x0, 0x0, 0x0})

	writer := bufio.NewWriter(mwd)

	_, err := writer.WriteString("Hello World!")
	err = writer.Flush()

	result := wd.String()

	if err != nil {
		t.Error(err)
	}

	if result != "Hello World!" {
		t.Error("Expected `result` to equal `Hello World!`")
	}
}

func TestWriterMasked(t *testing.T) {
	var input []byte

	underTest := []byte{0xa, 0xb, 0xc, 0xd, 0xe, 0xf, 0x1, 0x2}

	mask := [4]byte{0xf, 0x0, 0xf, 0x0}

	wd := bytes.NewBuffer([]byte{})
	mwd := NewMaskedWriter(wd, mask)

	copy(input, underTest)
	_, err := mwd.Write(input)

	if err != nil {
		t.Error(err)
	}

	bts := wd.Bytes()

	for i, b := range bts {
		m := mask[i%4]
		e := underTest[i] ^ m

		if e != b {
			t.Error("Expected e to equal b")
		}
	}
}
