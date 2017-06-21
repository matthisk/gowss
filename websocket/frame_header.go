package websocket

import (
	"encoding/binary"
)

//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-------+-+-------------+-------------------------------+
// |F|R|R|R| opcode|M| Payload len |    Extended payload length    |
// |I|S|S|S|  (4)  |A|     (7)     |             (16/64)           |
// |N|V|V|V|       |S|             |   (if payload len==126/127)   |
// | |1|2|3|       |K|             |                               |
// +-+-+-+-+-------+-+-------------+ - - - - - - - - - - - - - - - +
// |     Extended payload length continued, if payload len == 127  |
// + - - - - - - - - - - - - - - - +-------------------------------+
// |                               |Masking-key, if MASK set to 1  |
// +-------------------------------+-------------------------------+
// | Masking-key (continued)       |          Payload Data         |
// +-------------------------------- - - - - - - - - - - - - - - - +
// :                     Payload Data continued ...                :
// + - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - +
// |                     Payload Data continued ...                |
// +---------------------------------------------------------------+

// FrameHeader struct that captures a Websocket frame
type FrameHeader struct {
	final  bool
	opcode byte

	rsv1 bool
	rsv2 bool
	rsv3 bool

	mask          bool
	maskBytes     [4]byte
	payloadLength int
}

// NewFrameHeader can be used to construct a frame header where rsv control bits are set to 0
func NewFrameHeader(final bool, opcode byte, mask bool, maskBytes [4]byte, payloadLength int) FrameHeader {
	return FrameHeader{final, opcode, false, false, false, mask, maskBytes, payloadLength}
}

func (fh FrameHeader) toByteSlice() (result []byte) {
	var byte1 byte
	var byte2 byte

	if fh.final {
		byte1 = 1 << 7
	}

	if fh.rsv1 {
		byte1 |= 1 << 6
	}

	if fh.rsv2 {
		byte1 |= 1 << 5
	}

	if fh.rsv3 {
		byte1 |= 1 << 4
	}

	byte1 |= fh.opcode

	if fh.mask {
		byte2 = 1 << 7
	}

	var plBytes []byte

	switch {
	case fh.payloadLength > 65537:
		byte2 |= 127

		// We have to transform the payloadLength to a byte slice to append it
		// to the frame header result. If the payloadLength is greater than 65537
		// this means we have to store this length in a uint64
		plBytes = make([]byte, 8)
		binary.LittleEndian.PutUint64(plBytes, uint64(fh.payloadLength))
	case fh.payloadLength > 125:
		byte2 |= 126

		// We have to transform the payloadLength to a byte slice to append it
		// to the frame header result. If the payload length is greater than 125
		// but smaller than 65537 this means we have to store this length in a
		// uint16
		plBytes = make([]byte, 2)
		binary.LittleEndian.PutUint16(plBytes, uint16(fh.payloadLength))
	default:
		byte2 |= byte(fh.payloadLength)
	}

	result = append(result, byte1, byte2)
	result = append(result, plBytes...)

	if fh.mask {
		result = append(result, fh.maskBytes[0], fh.maskBytes[1], fh.maskBytes[2], fh.maskBytes[3])
	}

	return result
}
