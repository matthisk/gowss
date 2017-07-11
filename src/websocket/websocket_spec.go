package websocket

import (
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"
	"log"
)

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

// Closing status codes
const (
	websocketGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

	closeStatusNormal            = 1000
	closeStatusGoingAway         = 1001
	closeStatusProtocolError     = 1002
	closeStatusUnsupportedData   = 1003
	closeStatusFrameTooLarge     = 1004
	closeStatusNoStatusRcvd      = 1005
	closeStatusAbnormalClosure   = 1006
	closeStatusBadMessageData    = 1007
	closeStatusPolicyViolation   = 1008
	closeStatusTooBigData        = 1009
	closeStatusExtensionMismatch = 1010

	maxControlFramePayloadLength = 125
)

// The message types are defined in RFC 6455, section 11.8.
const (
	// ContinuationFrame denotes the continuation of a fragmented message.
	ContinuationFrame = 0

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

	// PongMessage denotes a pong control message. The optional message payload
	// is UTF-8 encoded text.
	PongMessage = 10
)

type FrameHandler interface {
	ReadMessage() (byte, []byte, error)
	WriteMessage(byte, []byte) error
	CloseConnection(int, string) error
	NextReader() (byte, io.Reader, error)
	NextWriter(byte, int64) (io.Writer, error)
}

// FrameSpecHandler handles websocket specification
type FrameSpecHandler struct {
	conn *Conn
}

// NewFrameSpecHandler creates a new frame specification handler
func NewFrameSpecHandler(conn *Conn) *FrameSpecHandler {
	return &FrameSpecHandler{conn}
}

func isControlFrameOpcode(opcode byte) bool {
	return opcode == CloseMessage || opcode == PingMessage || opcode == PongMessage
}

func isBadFrame(fh FrameHeader) error {
	switch fh.opcode {
	case ContinuationFrame:
		return ErrBadFrame
	}

	return nil
}

func isFragmentedFrameStart(fin bool, opcode byte) bool {
	return !fin && opcode != ContinuationFrame
}

func (fspec *FrameSpecHandler) receiveFrame() (fh FrameHeader, err error) {
	conn := fspec.conn
	p, err := conn.read(2)

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

	fh.payloadLength = payloadLength

	switch payloadLength {
	case 126:
		pl, err := conn.read(2)

		if err != nil {
			return fh, err
		}

		payloadLength = int64(binary.BigEndian.Uint16(pl))

		fh.payloadLength = payloadLength
	case 127:
		pl, err := conn.read(8)

		if err != nil {
			return fh, err
		}

		payloadLength = int64(binary.BigEndian.Uint64(pl))

		fh.payloadLength = payloadLength

		if pl[0] > 127 {
			return fh, errors.New("Most significant bit of payloadLength should be 0")
		}
	}

	// 3: Mask Bits

	if mask {
		maskingBytes, err = conn.read(4)

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

func (fspec *FrameSpecHandler) handleControlFrame(fh FrameHeader, reader io.Reader) error {
	conn := fspec.conn
	switch fh.opcode {
	case CloseMessage:
		conn.receivedClose = true
		return fspec.handleCloseMessage(fh, reader)
	case PingMessage:
		return fspec.handlePingMessage(fh, reader)
	case PongMessage:
		return fspec.handlePongMessage(fh, reader)
	}

	return nil
}

func (fspec *FrameSpecHandler) handlePongMessage(fh FrameHeader, reader io.Reader) error {
	log.Println("Received pong message, continue")
	return nil
}

func (fspec *FrameSpecHandler) handlePingMessage(fh FrameHeader, reader io.Reader) error {
	log.Println("Received ping message, now sending pong")
	payload, err := ioutil.ReadAll(reader)

	if err != nil {
		return err
	}

	err = fspec.WriteMessage(PongMessage, payload)

	return err
}

func (fspec *FrameSpecHandler) handleCloseMessage(fh FrameHeader, reader io.Reader) error {
	message, err := ioutil.ReadAll(reader)

	if err != nil {
		return err
	}

	// The first two bytes are an unsigned integer containing the
	// error code that explains why the socket was closed
	statusCode := binary.BigEndian.Uint16(message[0:2])
	statusMsg := string(message[2:])

	log.Println("Received CLOSE opcode with status:", statusCode, statusMsg)

	// After reading the payload we send a close message to the client
	// in case we haven't already sent this.

	if fspec.conn.sentClose {
		log.Println("Already sent a close message, not sending again")
		return errors.New("Received close message")
	}

	err = fspec.CloseConnection(int(statusCode), "Too bad man")

	if err == nil {
		fspec.conn.sentClose = true
		return err
	}

	return errors.New("Received close message")
}

// NextReader generate a reader for the next frame
func (fspec *FrameSpecHandler) NextReader() (opcode byte, r io.Reader, err error) {
	// 1 : Receive the frame header
	fh, err := fspec.receiveFrame()

	if err != nil {
		return fh.opcode, r, err
	}

	// 2 : Check if the frame header is valid

	if err := isBadFrame(fh); err != nil {
		// TODO: close the connection

		return fh.opcode, r, err
	}

	// 3 : Create a reader for the payload

	reader := io.LimitReader(fspec.conn.brw, fh.payloadLength)

	// 4 : Handle control frames

	if isControlFrameOpcode(fh.opcode) {
		if err := fspec.handleControlFrame(fh, reader); err != nil {
			return fh.opcode, r, err
		}

		// Control frames are not exposed to the libraries users
		// we thus continue by reading the next frame
		return fspec.NextReader()
	}

	// 5 : Handle fragmented messages

	if isFragmentedFrameStart(fh.final, fh.opcode) {
		reader = &fragmentReader{reader, fspec}
	}

	// 6 : Log & Return

	log.Println("Receiving Frame", fh.opcode, fh.payloadLength)

	return fh.opcode, reader, err
}

// NextWriter write the frameheader to the conn and return a masked writer
func (fspec *FrameSpecHandler) NextWriter(opcode byte, payloadLength int64) (w io.Writer, err error) {
	conn := fspec.conn
	fh := NewFrameHeader(false, opcode, !conn.isServer, conn.mask, payloadLength)

	_, err = conn.brw.Write(fh.toByteSlice())

	if err != nil {
		return w, err
	}

	err = conn.brw.Flush()

	if err != nil {
		return w, err
	}

	return NewMaskedWriter(conn.brw, fh.maskBytes), nil
}

// ReadMessage read all bytes in payload using ioutil.ReadAll
func (fspec *FrameSpecHandler) ReadMessage() (opcode byte, message []byte, err error) {
	opcode, reader, err := fspec.NextReader()

	if err != nil {
		return 0, message, err
	}

	message, err = ioutil.ReadAll(reader)

	return opcode, message, err
}

// WriteMessage write all bytes in payload to writer
func (fspec *FrameSpecHandler) WriteMessage(opcode byte, b []byte) (err error) {
	writer, err := fspec.NextWriter(opcode, int64(len(b)))

	if err != nil {
		return err
	}

	_, err = writer.Write(b)

	if err != nil {
		return err
	}

	// Write the message from the buffer to the connection
	fspec.conn.Flush()

	return nil
}

// CloseConnection sends the CloseMessage opcode to the receiver
func (fspec *FrameSpecHandler) CloseConnection(statusCode int, statusMessage string) (err error) {
	payload := make([]byte, 2)
	msg := []byte(statusMessage)

	binary.BigEndian.PutUint16(payload, uint16(statusCode))
	payload = append(payload, msg...)

	return fspec.WriteMessage(CloseMessage, payload)
}
