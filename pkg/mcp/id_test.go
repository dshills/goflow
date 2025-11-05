package mcp

import (
	"encoding/json"
	"testing"
)

func TestIDComparison(t *testing.T) {
	// When we send uint64(1), it marshals to JSON number 1
	// When we receive it back, it unmarshals to float64(1)
	// These won't be equal with ==

	reqID := uint64(1)

	// Marshal and unmarshal to simulate round-trip
	data, _ := json.Marshal(map[string]interface{}{"id": reqID})
	t.Logf("Marshaled: %s", string(data))

	var result map[string]interface{}
	json.Unmarshal(data, &result)

	respID := result["id"]
	t.Logf("Request ID type: %T, value: %v", reqID, reqID)
	t.Logf("Response ID type: %T, value: %v", respID, respID)

	if reqID == respID {
		t.Log("IDs are equal with ==")
	} else {
		t.Log("IDs are NOT equal with ==")
		t.Log("This is the problem!")
	}

	// Test with interface{} comparison
	var reqIDInterface interface{} = reqID
	if reqIDInterface == respID {
		t.Log("IDs are equal when both are interface{}")
	} else {
		t.Log("IDs still NOT equal when both are interface{}")
	}
}
