package main

import (
	"github.com/dshills/goflow/internal/testutil/mocks"
)

func main() {
	server := mocks.NewMockMCPServer()
	if err := server.Run(); err != nil {
		panic(err)
	}
}
