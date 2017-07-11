package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"./websocket"
)

func handleConnection(conn websocket.Conn) {
	for true {
		opcode, message, err := conn.Receive()

		if err != nil {
			fmt.Println("Failed to receive message", err)
			return
		}

		switch opcode {
		case websocket.TextMessage:
			msg := string(message)
			log.Println("Received text message:", msg)
		case websocket.BinaryMessage:
			log.Println("Received binary message:", message)
		default:
			log.Println("Received unknown message opcode:", opcode)
		}
	}
}

func runServer() error {
	server := websocket.CreateWSServer()

	defer server.Close()

	websocket.HandleFunc("/chat", handleConnection)

	if err := server.ListenAndServe(); err != nil {
		log.Println("Unable to create server", err)
		return err
	}

	return nil
}

func readClient(conn websocket.Conn) {
	for true {
		opcode, msg, err := conn.Receive()

		if err != nil {
			log.Println("Failed to read message with error", err)
			break
		}

		log.Println("Read message", opcode, msg)
	}
}

func runClient() {
	conn, err := websocket.Dial("ws://localhost:8080/chat")

	if err != nil {
		log.Println("Failed to open websocket connection", err)
		return
	}

	// Start consuming messages in a new go-routine
	go readClient(conn)

	// Send a couple of test messages and see how the server responds
	sentMessages := 0

	for true {
		if sentMessages > 1 {
			break
		}

		msg := []byte("Hello Websocket World!")

		err = conn.Send(websocket.TextMessage, msg)

		if err != nil {
			fmt.Println("Failed to write message to websocket", err)
			return
		}

		err := conn.Send(websocket.PingMessage, []byte("ping ping ping!"))

		if err != nil {
			log.Println("Failed to send ping message", err)
			return
		}

		time.Sleep(2 * time.Second)

		sentMessages++
	}

	// Send the close frame

	err = conn.Close()

	if err != nil {
		log.Println("Failed to close connection", err)
	}

	for true {
		if err = conn.Send(websocket.TextMessage, []byte("Is this possible?")); err != nil {
			log.Println("Unable to send with err", err)
			break
		}

		time.Sleep(5 * time.Second)
	}
}

func main() {
	log.Println("Start our websocket test")

	runCode := os.Args[1]

	if runCode == "server" {
		runServer()
	}

	if runCode == "client" {
		runClient()
	}
}
