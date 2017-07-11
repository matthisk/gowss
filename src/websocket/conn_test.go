package websocket

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"io"
	"log"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type PRStub struct {
	reader    io.Reader
	remaining int64
}

func (pr *PRStub) Len() int64 {
	return 0
}

func (pr *PRStub) PayloadType() byte {
	return TextMessage
}

func (pr *PRStub) Remaining() int64 {
	return pr.remaining
}

func (pr *PRStub) Read(b []byte) (n int, err error) {
	n, err = pr.reader.Read(b)

	pr.remaining -= int64(n)

	return n, err
}

type FrameHandlerStub struct {
	getPR func() PayloadReader
}

func (sp *FrameHandlerStub) ReadMessage() (byte, []byte, error) {
	return 0x0, []byte{}, nil
}

func (sp *FrameHandlerStub) WriteMessage(byte, []byte) error {
	return nil
}

func (sp *FrameHandlerStub) CloseConnection(int, string) error {
	return nil
}

func (sp *FrameHandlerStub) NextReader() (byte, io.Reader, error) {
	return 0x0, sp.getPR(), nil
}

func (sp *FrameHandlerStub) NextWriter(opcode byte, pl int64) (io.Writer, error) {
	return nil, nil
}

func NewFrameHandlerStub() *FrameHandlerStub {
	return &FrameHandlerStub{
		getPR: func() PayloadReader {
			reader := bytes.NewBuffer([]byte{0x0, 0x1, 0x2, 0x3})
			return &PRStub{reader, 4}
		},
	}
}

func wsPipe() (src net.Conn, ws Conn, err error) {
	src, dest := net.Pipe()

	r := bufio.NewReader(dest)
	w := bufio.NewWriter(dest)
	rw := bufio.NewReadWriter(r, w)

	request := &http.Request{}

	ws, err = NewConn(dest, rw, request)

	return src, ws, err
}

func TestConnRead(t *testing.T) {
	handler := NewFrameHandlerStub()

	conn := Conn{
		Handler: handler,
		pr:      nil,
	}

	read := func() {
		buf := make([]byte, 4)

		n, err := conn.Read(buf)

		assert.Nil(t, err)
		assert.Equal(t, 4, n, "Expected to read 4 bytes")

		assert.Equal(t, []byte{0x0, 0x1, 0x2, 0x3}, buf, "Expected to read exact byte slice")
	}

	read()
	read()
}

func TestConnWrite(t *testing.T) {
	handler := NewFrameHandlerStub()

	conn := Conn{
		Handler: handler,
	}

	write := func() {
		buf := []byte{0x0, 0x1, 0x2, 0x3}
		n, err := conn.Write(buf)

		assert.Nil(t, err)
		assert.Equal(t, 4, n, "Expected to write 4 bytes")
	}

	write()
}

func TestWSConnRead(t *testing.T) {
	// 1: Create test connections src -> dest

	src, ws, err := wsPipe()

	assert.Nil(t, err)

	// 2: Write to src socket

	write := func(msg string) {
		payload := []byte(msg)
		fh := NewFrameHeader(false, TextMessage, true, [4]byte{0x5, 0xa, 0xd, 0x1}, int64(len(payload)))

		_, err = src.Write(fh.toByteSlice())

		assert.Nil(t, err)

		writer := NewMaskedWriter(src, fh.maskBytes)
		_, err = writer.Write(payload)

		assert.Nil(t, err)
	}

	go write("Hello i am testing this socket with a message!")

	time.Sleep(500 * time.Millisecond)

	go write("Second message!")

	// 3 : Read from websocket conn

	read := func(i int) (string, error) {
		buf := make([]byte, i)
		n, err := ws.Read(buf)

		assert.Nil(t, err)

		return string(buf[:n]), nil
	}

	r1, err := read(5)
	assert.Nil(t, err)
	r2, err := read(1)
	assert.Nil(t, err)
	r3, err := read(1)
	assert.Nil(t, err)
	r4, err := read(100)
	assert.Nil(t, err)

	assert.Equal(t, "Hello", r1, "Unexpected payload read")
	assert.Equal(t, " ", r2, "Unexpected payload read")
	assert.Equal(t, "i", r3, "Unexpected payload read")
	assert.Equal(t, " am testing this socket with a message!", r4, "Unexpected payload read")

	s, err := read(25)

	assert.Nil(t, err)
	assert.Equal(t, 15, len(s), "Read too many bytes from frame")
}

func TestWSConnWrite(t *testing.T) {
	consumer, ws, err := wsPipe()

	assert.Nil(t, err)

	write := func(opcode byte, msg []byte) {
		ws.FrameType = opcode

		_, err := ws.Write(msg)

		assert.Nil(t, err)
	}

	read := func(expectedOpCode byte) {
		header := make([]byte, 2)

		_, err = consumer.Read(header)

		assert.Nil(t, err)

		opcode := header[0] & 0xf

		assert.Equal(t, expectedOpCode, opcode, "Unexpected opcode for frame")
	}

	go write(TextMessage, []byte("Hello world!"))

	time.Sleep(500 * time.Millisecond)

	go write(BinaryMessage, []byte{0x1, 0x2, 0x3, 0x4})

	read(TextMessage)
	read(BinaryMessage)
}

func TestWSWriteMediumMessage(t *testing.T) {
	conn, ws, err := wsPipe()

	// 1 : Generate large byte slice of length 2^15

	l := 1<<16 - 1
	b := make([]byte, l)

	n, err := rand.Read(b)

	assert.Nil(t, err)
	assert.Equal(t, l, n, "Read less bytes than expected")

	// 2 : Write this large payload to the websocket

	write := func() {
		ws.FrameType = BinaryMessage
		_, err = ws.Write(b)

		assert.Nil(t, err)
	}

	go write()

	// 3 : Read the frameheader from the conn

	read := func(l int) []byte {
		buf := make([]byte, l)

		n, err := conn.Read(buf)

		assert.Nil(t, err)
		assert.Equal(t, l, n, "Read less bytes than expected")

		return buf
	}

	byte1 := read(1)[0]
	byte2 := read(1)[0]

	opcode := byte1 & 0xf

	assert.Equal(t, uint8(BinaryMessage), opcode, "Expected opcode for binary msg")

	assert.Equal(t, uint8(126), byte2&0x7f, "Expected payload length of 126")

	assert.Equal(t, uint8(0x0), byte2&(1<<7), "Expected mask bit to be 0")

	byte34 := read(2)

	payloadLength := int(binary.LittleEndian.Uint16(byte34))

	assert.Equal(t, l, payloadLength, "Wrong payload length parsed")

	payload := read(payloadLength)

	assert.Equal(t, b, payload, "Wrong payload")
}

func TestWSWriteLargeMessage(t *testing.T) {
	conn, ws, err := wsPipe()

	// 1 : Generate a writer for a large payload size

	var pl int64 = 1 << 20

	wr, err := ws.Handler.NextWriter(BinaryMessage, pl)

	if err != nil {
		t.Error("Failed to create writer for frame", err)
		return
	}

	// 2 : Write pl amount of bytes into the connection (streaming)

	write := func() {
		limitRand := io.LimitReader(rand.Reader, pl)

		written, err := io.CopyBuffer(wr, limitRand, nil)

		assert.Nil(t, err)

		log.Println("written", written)

		ws.Flush()
	}

	go write()

	// 3 : Read from conn

	read := func(l int) []byte {
		buf := make([]byte, l)

		n, err := conn.Read(buf)

		assert.Equal(t, n, l, "Expected to read l bytes")
		assert.Nil(t, err)

		return buf
	}

	header := read(10)

	payloadLength := binary.LittleEndian.Uint64(header[2:])

	assert.Equal(t, payloadLength, pl, "Expected to receive pyaload length amount of bytes")

	bytesRem := int64(payloadLength)

	buf := make([]byte, 30*1024)
	for {
		if bytesRem <= 0 {
			break
		}

		n, err := conn.Read(buf)

		assert.Nil(t, err)

		bytesRem -= int64(n)
	}

	log.Println("Read all from connection")
}

func TestPingControlMessage(t *testing.T) {
	conn, ws, err := wsPipe()

	assert.Nil(t, err)

	// Write PING message

	pl := []byte("This is a ping message!")
	fh := NewFrameHeader(false, PingMessage, true, [4]byte{0x0, 0x0, 0x0, 0x0}, int64(len(pl)))

	write := func() {
		conn.Write(fh.toByteSlice())
		conn.Write(pl)
	}

	go write()

	// Read from WS

	read := func() {
		opcode, msg, err := ws.Receive()

		assert.Nil(t, err)

		log.Println("Received", opcode, msg)
	}

	go read()

	// Read from conn

	header := make([]byte, 2)
	_, err = conn.Read(header)

	assert.Nil(t, err)

	opcode := header[0] & 0xf
	payloadLength := header[1] & 0x7f

	assert.Equal(t, uint8(PongMessage), opcode, "Expected to receive PongMessage")
	assert.Equal(t, uint8(len(pl)), payloadLength, "Expected to receive same pl length")

	payload := make([]byte, len(pl))
	_, err = conn.Read(payload)

	assert.Nil(t, err)

	assert.Equal(t, pl, payload, "Expected to receive same payload")
}
