package websocket

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"time"
)

var (
	errSetDeadline = errors.New("Error setting read/write deadline")
)

// Conn struct to resemble a websocket connection
type Conn struct {
	rwc       io.ReadWriteCloser
	Handler   FrameHandler
	FrameType byte
	request   *http.Request

	// State
	isServer      bool
	receivedClose bool
	sentClose     bool

	// Writing Specific
	mask [4]byte

	// ReaderWriter
	pr  io.Reader
	brw *bufio.ReadWriter
}

// SetDeadline sets the read and write deadline on underlying network connection
func (conn *Conn) SetDeadline(t time.Time) error {
	if conn, ok := conn.rwc.(net.Conn); ok {
		return conn.SetDeadline(t)
	}

	return errSetDeadline
}

// SetReadDeadline sets the read deadline on underlying network connection
func (conn *Conn) SetReadDeadline(t time.Time) error {
	if conn, ok := conn.rwc.(net.Conn); ok {
		return conn.SetReadDeadline(t)
	}

	return errSetDeadline
}

// SetWriteDeadline sets the write deadline on underlying network connection
func (conn *Conn) SetWriteDeadline(t time.Time) error {
	if conn, ok := conn.rwc.(net.Conn); ok {
		return conn.SetWriteDeadline(t)
	}

	return errSetDeadline
}

// Close closes the underlying network connection
func (conn *Conn) Close() error {
	if err := conn.Handler.CloseConnection(closeStatusNormal, "closing connection"); err != nil {
		return err
	}

	return conn.rwc.Close()
}

// Read read from the websocket
func (conn *Conn) Read(b []byte) (n int, err error) {
	n, err = conn.pr.Read(b)

	// if err == io.EOF {
	// 	opcode, reader, err = conn.Handler.NextReader()

	// 	if err != nil {
	// 		return n, err
	// 	}

	// 	conn.pr = reader

	// 	m, err := conn.Read(b[n:])

	// 	return m + n, err
	// }

	return n, err
}

// Write to the websocket connection
func (conn *Conn) Write(b []byte) (n int, err error) {
	err = conn.Handler.WriteMessage(conn.FrameType, b)

	if err != nil {
		return 0, err
	}

	return len(b), nil
}

// Receive reads one message frame with an opcode Text / Binary from the websocket connection
func (conn *Conn) Receive() (byte, []byte, error) {
	return conn.Handler.ReadMessage()
}

// Send sends one message with opcode on the webscoket connection
func (conn *Conn) Send(opcode byte, b []byte) error {
	return conn.Handler.WriteMessage(opcode, b)
}

// Flush flush the underlying buffered writer
func (conn *Conn) Flush() error {
	return conn.brw.Flush()
}

// Read n bytes from the buffered ReadWriter
func (conn *Conn) read(n int) ([]byte, error) {
	result, err := conn.brw.Peek(n)

	if err != nil {
		fmt.Println("Error while peeking read buffer", err)
		return result, err
	}

	_, err = conn.brw.Discard(n)

	if err != nil {
		fmt.Println("Error while discarding read buffer", err)
	}

	return result, err
}

// NewConn return a new websocket connection from a net.Conn
func NewConn(conn net.Conn, bufrw *bufio.ReadWriter, request *http.Request) (c Conn, err error) {
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

	result := Conn{
		rwc:           conn,
		request:       request,
		isServer:      request != nil,
		receivedClose: false,
		sentClose:     false,
		mask:          mask,
		brw:           bufrw,
	}

	result.Handler = NewFrameSpecHandler(&result)

	return result, nil
}
