package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/xeipuuv/gojsonschema"
)

// ValidateAgainstSchema validates workflow YAML bytes against the JSON schema
func ValidateAgainstSchema(yamlBytes []byte) error {
	if len(yamlBytes) == 0 {
		return errors.New("empty YAML input")
	}

	// Parse YAML into a generic structure that can be validated
	// gojsonschema can work with Go data structures
	var data interface{}

	// Try to parse as YAML first (since that's what we expect)
	wf, err := Parse(yamlBytes)
	if err != nil {
		return fmt.Errorf("failed to parse YAML for validation: %w", err)
	}

	// Convert workflow to JSON for validation (schema validator works with JSON)
	jsonBytes, err := json.Marshal(wf)
	if err != nil {
		return fmt.Errorf("failed to convert workflow to JSON for validation: %w", err)
	}

	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return fmt.Errorf("failed to unmarshal workflow JSON: %w", err)
	}

	// Load the schema
	schemaPath := "specs/001-goflow-spec-review/contracts/workflow-schema-v1.json"
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	// Create schema loader
	schemaLoader := gojsonschema.NewBytesLoader(schemaBytes)
	documentLoader := gojsonschema.NewGoLoader(data)

	// Validate
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("schema validation error: %w", err)
	}

	if !result.Valid() {
		// Collect all validation errors
		var errMsg string
		for i, desc := range result.Errors() {
			if i > 0 {
				errMsg += "; "
			}
			errMsg += fmt.Sprintf("%s: %s", desc.Field(), desc.Description())
		}
		return fmt.Errorf("schema validation failed: %s", errMsg)
	}

	return nil
}

// TopologicalSort performs a topological sort on the workflow nodes
// Returns an ordered list of node IDs that respects the dependency order
func TopologicalSort(workflow *Workflow) ([]NodeID, error) {
	if workflow == nil {
		return nil, errors.New("workflow cannot be nil")
	}

	// Build adjacency list and in-degree map
	adjacency := make(map[string][]string)
	inDegree := make(map[string]int)

	// Initialize in-degree for all nodes
	for _, node := range workflow.Nodes {
		nodeID := node.GetID()
		inDegree[nodeID] = 0
		adjacency[nodeID] = []string{}
	}

	// Build adjacency list and calculate in-degrees
	for _, edge := range workflow.Edges {
		adjacency[edge.FromNodeID] = append(adjacency[edge.FromNodeID], edge.ToNodeID)
		inDegree[edge.ToNodeID]++
	}

	// Kahn's algorithm: start with nodes that have no incoming edges
	queue := make([]string, 0)
	for _, node := range workflow.Nodes {
		nodeID := node.GetID()
		if inDegree[nodeID] == 0 {
			queue = append(queue, nodeID)
		}
	}

	// Process nodes in topological order
	result := make([]NodeID, 0, len(workflow.Nodes))
	for len(queue) > 0 {
		// Remove node from queue
		current := queue[0]
		queue = queue[1:]
		result = append(result, NodeID(current))

		// For each neighbor, reduce in-degree
		for _, neighbor := range adjacency[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// If we haven't processed all nodes, there's a cycle
	if len(result) != len(workflow.Nodes) {
		return nil, errors.New("workflow contains a cycle (circular dependency)")
	}

	return result, nil
}
