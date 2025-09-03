package main

import "dungtl2003/chat-app-message-service/internal/server"

func main() {
	server, err := server.New(nil)
	if err != nil {
		panic(err)
	}

	err = server.Run()
	if err != nil {
		panic(err)
	}
}
