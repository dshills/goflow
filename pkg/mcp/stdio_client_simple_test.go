package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestMockServerRaw(t *testing.T) {
	// Test the test server directly to ensure it works
	mockServerPath, err := filepath.Abs("../../cmd/testserver/main.go")
	if err != nil {
		t.Fatalf("Failed to get test server path: %v", err)
	}

	cmd := exec.Command("go", "run", mockServerPath)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to create stdin: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to create stdout: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer cmd.Process.Kill()

	// Send initialize
	initReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "test",
				"version": "0.1.0",
			},
		},
	}

	reqJSON, _ := json.Marshal(initReq)
	fmt.Fprintf(stdin, "%s\n", reqJSON)

	// Read response
	scanner := bufio.NewScanner(stdout)
	done := make(chan bool)
	var respLine string

	go func() {
		if scanner.Scan() {
			respLine = scanner.Text()
			done <- true
		}
	}()

	select {
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for response")
	case <-done:
		t.Logf("Got response: %s", respLine)

		var resp JSONRPCResponse
		if err := json.Unmarshal([]byte(respLine), &resp); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if resp.Error != nil {
			t.Fatalf("Got error response: %v", resp.Error)
		}
	}
}

func TestJSONRPCMarshaling(t *testing.T) {
	// Test that our request format is correct
	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      uint64(1),
		Method:  "initialize",
		Params:  json.RawMessage(`{"test":"value"}`),
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	t.Logf("Marshaled request: %s", string(data))

	// Verify it unmarshals correctly
	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded["jsonrpc"] != "2.0" {
		t.Errorf("Expected jsonrpc 2.0, got %v", decoded["jsonrpc"])
	}
}

func TestResponseParsing(t *testing.T) {
	// Test parsing the mock server's response format
	mockResp := `{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"tools":{}},"protocolVersion":"2024-11-05","serverInfo":{"name":"mock-mcp-server","version":"0.1.0"}}}`

	var resp JSONRPCResponse
	if err := json.Unmarshal([]byte(mockResp), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("Unexpected error in response: %v", resp.Error)
	}

	if len(resp.Result) == 0 {
		t.Fatal("Expected non-empty result")
	}

	t.Logf("Parsed result: %s", string(resp.Result))
}

func TestScannerBehavior(t *testing.T) {
	// Test that bufio.Scanner works as expected with our format
	input := "line1\nline2\nline3\n"
	scanner := bufio.NewScanner(bytes.NewReader([]byte(input)))

	lines := []string{}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	done := make(chan bool)
	go func() {
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		done <- true
	}()

	select {
	case <-ctx.Done():
		// Expected to timeout since scanner blocks
	case <-done:
		t.Logf("Read %d lines", len(lines))
	}

	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}
}
