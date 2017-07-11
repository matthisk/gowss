package websocket

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
)

// Dial open a websocket connection
func Dial(url string) (Conn, error) {
	return createClient(url)
}

func createWSSRequest(url string) (*http.Request, error) {
	request, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return request, err
	}

	request.Header.Set("Origin", "localhost")
	request.Header.Set("Upgrade", "websocket")
	request.Header.Set("Connection", "Upgrade")
	request.Header.Set("Sec-WebSocket-Key", "AQIDBAUGBwgJCgsMDQ4PEC==")
	request.Header.Set("Sec-WebSocket-Version", "13")

	return request, nil
}

// RunClient run a ws client
func createClient(inputURL string) (wsConn Conn, err error) {
	parsedURL, err := url.Parse(inputURL)

	if err != nil {
		return wsConn, err
	}

	address := parsedURL.Host
	fmt.Println("Connecting to", address, "ws server")

	request, err := createWSSRequest(inputURL)

	if err != nil {
		log.Println("Failed handshake", err)
		return wsConn, err
	}

	conn, err := net.Dial("tcp", address)

	if err != nil {
		log.Println("Failed to open TCP connection to client", err)
		return wsConn, err
	}

	writer := bufio.NewWriter(conn)
	reader := bufio.NewReader(conn)

	// Write the request handshake
	request.Write(writer)
	writer.Flush()

	response, err := http.ReadResponse(reader, request)

	if err != nil {
		log.Println("Failed to perform handshake", err)
		return wsConn, err
	}

	if response.StatusCode != 101 {
		err = fmt.Errorf("Wrong http status code %d", response.StatusCode)
		log.Println("Failed to perform handshake", err)
		return wsConn, err
	}

	return createWSSConn(conn)
}

func createWSSConn(conn net.Conn) (Conn, error) {
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	bufrw := bufio.NewReadWriter(r, w)

	wConn, err := NewConn(conn, bufrw, nil)

	if err != nil {
		fmt.Println("Failed to create websocket connection in client", err)
		return wConn, err
	}

	return wConn, nil
}
