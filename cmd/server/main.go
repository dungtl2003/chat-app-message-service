package main

import "dungtl2003/chat-app-message-service/internal/server"

func main() {
	server := server.New()
	server.Run()
}
