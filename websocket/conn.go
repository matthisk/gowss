package websocket

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
)

var ILLEGAL_OPCODE_ERR = errors.New("Expected opcode equal to TextMessage or BinaryMessage")

// The message types are defined in RFC 6455, section 11.8.
const (
	// TextMessage denotes a text data message. The text message payload is
	// interpreted as UTF-8 encoded text data.
	TextMessage = 1

	// BinaryMessage denotes a binary data message.
	BinaryMessage = 2

	// CloseMessage denotes a close control message. The optional message
	// payload contains a numeric code and text. Use the FormatCloseMessage
	// function to format a close message payload.
	CloseMessage = 8

	// PingMessage denotes a ping control message. The optional message payload
	// is UTF-8 encoded text.
	PingMessage = 9

	// PongMessage denotes a ping control message. The optional message payload
	// is UTF-8 encoded text.
	PongMessage = 10
)

// Conn struct to resemble a websocket connection
type Conn struct {
	conn net.Conn

	// Writing
	mask [4]byte
	bw   *bufio.Writer

	// Reading
	br *bufio.Reader
}

// NewConn return a new websocket connection from a net.Conn
func NewConn(conn net.Conn) (c Conn, err error) {
	var mask [4]byte
	maskSlice := make([]byte, 4)
	n, err := rand.Read(maskSlice)

	copy(mask[:], maskSlice[0:4])

	if err != nil {
		return c, err
	}

	if n != 4 {
		return c, errors.New("Expected 4 random bytes for the mask")
	}

	return Conn{conn, mask, bufio.NewWriter(conn), bufio.NewReader(conn)}, nil
}

// Bit masks used to parse control bits from frame header
const (
	// The first bit of the first byte denotes if this is the final frame
	finalBitMask = 1 << 7

	// The next three bits are reserved control bits, they should be 0
	rsv1BitMask = 1 << 6
	rsv2BitMask = 1 << 5
	rsv3BitMask = 1 << 4

	// The four bits after that denote which opcode this frame has (see const below for different opcodes)
	opCodeMask = 0xf

	// The next byte consists of one bit denoting if the payload is masked and 7 bits
	// telling us the size of the payload in bytes
	maskMask          = 1 << 7
	payloadLengthMask = 0x7f
)

func (conn *Conn) Read(l int) ([]byte, error) {
	result, err := conn.br.Peek(l)

	if err != nil {
		fmt.Println("Error while peeking read buffer", err)
		return result, err
	}

	_, err = conn.br.Discard(l)

	if err != nil {
		fmt.Println("Error while discarding read buffer", err)
	}

	return result, err
}

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
func (conn *Conn) receiveFrame() (fh FrameHeader, err error) {
	fmt.Println("Receive Frame")

	p, err := conn.Read(2)

	if err != nil {
		return fh, err
	}

	byte1 := p[0]
	byte2 := p[1]

	var maskingBytes []byte

	// 1: Control Bits

	final := byte1&finalBitMask != 0
	frameType := byte1 & opCodeMask
	mask := byte2&maskMask != 0

	fh.final = final
	fh.opcode = frameType
	fh.mask = mask

	// 2: Payload Length

	payloadLength := int64(byte2 & payloadLengthMask)

	fh.payloadLength = int(payloadLength)

	switch payloadLength {
	case 126:
		pl, err := conn.Read(2)

		if err != nil {
			return fh, err
		}

		payloadLength = int64(binary.BigEndian.Uint16(pl))

		fh.payloadLength = int(payloadLength)
	case 127:
		pl, err := conn.Read(8)

		if err != nil {
			return fh, err
		}

		payloadLength = int64(binary.BigEndian.Uint64(pl))

		fh.payloadLength = int(payloadLength)

		if pl[0] > 127 {
			return fh, errors.New("Most significant bit of payloadLength should be 0")
		}
	}

	// 3: Mask Bits

	if mask {
		maskingBytes, err = conn.Read(4)

		var maskBytes [4]byte
		copy(maskBytes[:], maskingBytes[0:4])

		fh.maskBytes = maskBytes

		if err != nil {
			return fh, err
		}

	}

	// 4: Return everything

	return fh, nil
}

// NextReader generate a reader for the next frame
func (conn *Conn) NextReader() (opcode byte, r *PayloadReader, err error) {
	fh, err := conn.receiveFrame()

	if err != nil {
		return fh.opcode, r, err
	}

	reader := NewPayloadReader(conn.br, fh)

	return fh.opcode, &reader, err
}

// ReadMessage read all bytes in payload using ioutil.ReadAll
func (conn *Conn) ReadMessage() (opcode byte, message []byte, err error) {
	opcode, reader, err := conn.NextReader()

	if err != nil {
		return 0, message, err
	}

	message, err = ioutil.ReadAll(reader)

	return opcode, message, err
}

// WriteMessage write all bytes in payload to writer
func (conn *Conn) WriteMessage(opcode byte, b []byte) (n int, err error) {
	if opcode != TextMessage && opcode != BinaryMessage {
		return 0, ILLEGAL_OPCODE_ERR
	}

	fh := NewFrameHeader(false, opcode, true, conn.mask, len(b))

	// The masked writer should write directly on the conn and
	//not on the buffered writer, because we have no way to flush it
	maskedWriter := NewMaskedWriter(conn.conn, fh.maskBytes)

	n, err = conn.bw.Write(fh.toByteSlice())

	if err != nil {
		return n, err
	}

	err = conn.bw.Flush()

	if err != nil {
		return n, err
	}

	n, err = maskedWriter.Write(b)

	if err != nil {
		return n, err
	}

	return n, nil
}