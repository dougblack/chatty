package main

import (
	"github.com/dougblack/chatty"
)

func main() {
	server, err := chat.NewServer(8080)
	defer server.Stop()
	if err != nil {
		panic(server)
	}
	server.Start()
}
