package websocket

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
)

// Struct to resemble a websocket connection
type Conn struct {
	conn net.Conn

	// Writing
	bw *bufio.Writer

	// Reading
	br *bufio.Reader
}

// Struct that captures a Websocket frame
type FrameHeader struct {
	final         bool
	opcode        byte
	mask          bool
	maskBytes     []byte
	payloadLength int
}

const WEBSOCKET_HEADER_GUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

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

func (c *Conn) Read(l int) ([]byte, error) {
	result, err := c.br.Peek(l)

	if err != nil {
		fmt.Println("Error while peeking read buffer", err)
		return result, err
	}

	_, err = c.br.Discard(l)

	if err != nil {
		fmt.Println("Error while discarding read buffer", err)
	}

	return result, err
}

func createWebsocketSecHeader(input string) string {
	hasher := sha1.New()

	io.WriteString(hasher, input)
	io.WriteString(hasher, WEBSOCKET_HEADER_GUID)

	return base64.URLEncoding.EncodeToString(hasher.Sum(nil))
}

func validateRequest(request *http.Request) bool {
	return true
}

func badRequest() *http.Response {
	response := http.Response{Status: "400 BAD REQUEST", StatusCode: 400}

	return &response
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
func receiveFrame(conn Conn) (fh FrameHeader, err error) {
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

	// 2: Payload Length

	payloadLength := int64(byte2 & payloadLengthMask)

	switch payloadLength {
	case 126:
		pl, err := conn.Read(2)

		if err != nil {
			return fh, err
		}

		payloadLength = int64(binary.BigEndian.Uint16(pl))
	case 127:
		pl, err := conn.Read(8)

		if err != nil {
			return fh, err
		}

		payloadLength = int64(binary.BigEndian.Uint64(pl))

		if pl[0] > 127 {
			return fh, errors.New("Most significant bit of payloadLength should be 0")
		}
	}

	// 3: Mask Bits

	if mask {
		maskingBytes, err = conn.Read(4)

		if err != nil {
			return fh, err
		}
	}

	// 4: Payload

	// We only support 32 bit payloadLength on 32bit systems by typecasting to int from int64
	// payload, err := conn.Read(int(payloadLength))

	// 5: Return everything

	return FrameHeader{final, frameType, mask, maskingBytes, int(payloadLength)}, nil
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	request, err := http.ReadRequest(reader)

	if err != nil {
		log.Println("Failed to parse request:", err)
		badRequest().Write(writer)
		return
	}

	validRequest := validateRequest(request)

	if !validRequest {
		log.Println("Invalid request:", request)
		badRequest().Write(writer)
		return
	}

	headers := make(map[string][]string)
	headers["Upgrade"] = []string{"websocket"}
	headers["Connection"] = []string{"Upgrade"}
	headers["Sec-Websocket-Accept"] = []string{createWebsocketSecHeader(request.Header.Get("Sec-Websocket-Key"))}

	response := http.Response{Status: "101 Switching Protocols", StatusCode: 101, Header: headers}

	response.Write(writer)
	writer.Flush()

	fmt.Println("Received HTTP Request:", request)

	wConn := Conn{conn, bufio.NewWriter(conn), bufio.NewReader(conn)}

	_, err = receiveFrame(wConn)

	if err != nil {
		fmt.Println(err)
	}
}

func main() {
	ln, err := net.Listen("tcp", ":8080")
	defer ln.Close()

	fmt.Println("Websocket server listening on port 8080")

	if err != nil {
		log.Fatal(err)
		return
	}

	for {
		conn, err := ln.Accept()

		fmt.Println("Accepted connection")

		if err != nil {
			log.Fatal("Failed to accept connection:", err)
			continue
		}

		go handleConnection(conn)
	}
}
