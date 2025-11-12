package main

import (
	"log"

	"github.com/dshills/goflow/internal/testutil/testserver"
)

// main is the entry point when running the test MCP server as a standalone executable.
// This is used by integration tests to start an MCP server process via stdio transport.
func main() {
	// Load configuration from environment variables
	config := testserver.LoadConfig()

	// Create server instance
	server, err := testserver.NewServer(config)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start server (blocks until stdin is closed or error occurs)
	if err := server.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
