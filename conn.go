package main

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"time"
)

const input = `GET /chat HTTP/1.1
Host: example.com:8000
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==
Sec-WebSocket-Version: 13


`

func main() {
	fmt.Println("Connecting to localhost:8080 ws server")

	conn, err := net.Dial("tcp", "localhost:8080")

	if err != nil {
		fmt.Println(err)
		return
	}

	writer := bufio.NewWriter(conn)
	reader := bufio.NewReader(conn)

	_, err = writer.WriteString(input)

	if err != nil {
		fmt.Println(err)
		return
	}

	writer.Flush()

	response, err := http.ReadResponse(reader, nil)

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(response)

	payload := []byte{0xf, 0x82, 0x1, 0x1, 0x1, 0x1, 0xff, 0xff}

	writer.Write(payload)

	time.Sleep(time.Duration(1) * time.Second)

	writer.Flush()

	for true {

	}
}
