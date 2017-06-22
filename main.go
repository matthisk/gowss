package main

import (
	"fmt"
	"log"
	"os"

	"./websocket"
)

func handleConnection(conn websocket.Conn) {
	for true {
		opcode, message, err := conn.ReadMessage()

		if err != nil {
			fmt.Println("Failed to receive message", err)
			return
		}

		switch opcode {
		case websocket.TextMessage:
			msg := string(message)
			fmt.Println("Received text message:", msg)
		case websocket.BinaryMessage:
			fmt.Println("Received binary message:", message)
		default:
			fmt.Println("Received unknown message opcode:", opcode)
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

func main() {
	log.Println("Start our websocket test")

	runCode := os.Args[1]

	if runCode == "server" {
		runServer()
	}

	if runCode == "client" {
		websocket.RunClient()
	}
}
