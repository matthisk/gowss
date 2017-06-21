package main

import (
	"fmt"
	"os"

	"./websocket"
)

func main() {
	fmt.Println("Start our websocket test")

	runCode := os.Args[1]

	if runCode == "server" {
		websocket.RunServer()
	}

	if runCode == "client" {
		websocket.RunClient()
	}
}
