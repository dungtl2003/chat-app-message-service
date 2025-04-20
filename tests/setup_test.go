package tests

import (
	"dungtl2003/chat-app-message-service/internal/server"
	"fmt"
	"testing"
)

func setup(server *server.Server) {
	fmt.Println("Run setup")
	go func() {
		server.Run()
	}()
}

func teardown(server *server.Server) {
	fmt.Println("Run teardown")
	server.Close()
}

func TestMain(m *testing.M) {
	server := server.New()
	setup(server)
	m.Run()
	teardown(server)
}
