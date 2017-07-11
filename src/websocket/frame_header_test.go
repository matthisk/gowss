package websocket

import (
	"bytes"
	"testing"
)

func TestFrameHeader(t *testing.T) {
	fh := NewFrameHeader(false, 0x5, true, [4]byte{0xf, 0xf, 0xf, 0xf}, 100)

	bs := fh.toByteSlice()

	if len(bs) != 6 {
		t.Errorf("Expected bs to have length 6 but got %d", len(bs))
	}

	if final := bs[0] >> 7; final != 0 {
		t.Errorf("Expected first bit to be 0 but got 1")
	}

	if rsv := (bs[0] << 1) >> 5; rsv != 0 {
		t.Errorf("Expected reserved control bits to be set to 0 but got %d", rsv)
	}

	if opcode := (bs[0] << 4) >> 4; opcode != 0x5 {
		t.Errorf("Expected opcode to equal 5 but got %d", opcode)
	}

	if mask := bs[1] >> 7; mask != 1 {
		t.Errorf("Expected mask to equal 1 but got %d", mask)
	}

	if payloadLength := (bs[1] << 1) >> 1; payloadLength != 100 {
		t.Errorf("Expected payloadLength to equal 100 but got %d", payloadLength)
	}

	if maskBytes := bs[2:6]; !bytes.Equal(maskBytes, []byte{0xf, 0xf, 0xf, 0xf}) {
		t.Errorf("Expected maskBytes to equal {0xf, 0xf, 0xf, 0xf} but got %s", maskBytes)
	}
}

func TestFrameHeaderTwo(t *testing.T) {
	fh := NewFrameHeader(true, 0x9, false, [4]byte{0xf, 0xf, 0xf, 0xf}, 100)

	bs := fh.toByteSlice()

	if len(bs) != 2 {
		t.Errorf("Expected bs to have length 2 but got %d", len(bs))
	}

	if final := bs[0] >> 7; final != 1 {
		t.Errorf("Expected first bit to be 1 but got 1")
	}

	if rsv := (bs[0] << 1) >> 5; rsv != 0 {
		t.Errorf("Expected reserved control bits to be set to 0 but got %d", rsv)
	}

	if opcode := (bs[0] << 4) >> 4; opcode != 0x9 {
		t.Errorf("Expected opcode to equal 0x9 but got %d", opcode)
	}

	if mask := bs[1] >> 7; mask != 0 {
		t.Errorf("Expected mask to equal 0 but got %d", mask)
	}

	if payloadLength := (bs[1] << 1) >> 1; payloadLength != 100 {
		t.Errorf("Expected payloadLength to equal 100 but got %d", payloadLength)
	}
}

func TestFrameHeaderPayloadLengthTwo(t *testing.T) {
	fh := NewFrameHeader(false, 0x5, true, [4]byte{0xf, 0xf, 0xf, 0xf}, 126)
	bs := fh.toByteSlice()

	if len(bs) != 8 {
		t.Errorf("Expected bs to have length 8 but got %d", len(bs))
	}
}

func TestFrameHeaderPayloadLengthThree(t *testing.T) {
	fh := NewFrameHeader(false, 0x5, true, [4]byte{0xf, 0xf, 0xf, 0xf}, 127)
	bs := fh.toByteSlice()

	if len(bs) != 8 {
		t.Errorf("Expected bs to have length 8 but got %d", len(bs))
	}
}

func TestFrameHeader4(t *testing.T) {
	fh := NewFrameHeader(false, 0x5, true, [4]byte{0xf, 0xf, 0xf, 0xf}, 65538)
	bs := fh.toByteSlice()

	if len(bs) != 14 {
		t.Errorf("Expected bs to have length 14 but got %d", len(bs))
	}
}
