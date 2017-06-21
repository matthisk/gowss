package websocket

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

// RunClient
func RunClient() {
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

	wConn, err := NewConn(conn)

	if err != nil {
		fmt.Println("Failed to create websocket connection in client", err)
		return
	}

	for true {
		msg := []byte("Hello Websocket World!")

		_, err = wConn.WriteMessage(TextMessage, msg)

		if err != nil {
			fmt.Println("Failed to write message to websocket", err)
			return
		}

		time.Sleep(5 * time.Second)
	}
}
