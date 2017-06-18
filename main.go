package websocket

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
)

const WEBSOCKET_HEADER_GUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

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

	message, err := wConn.ReadMessage()

	if err != nil {
		fmt.Println("Failed to receive message", err)
	}

	fmt.Println("Message", message)
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
