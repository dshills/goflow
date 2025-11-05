package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Node is the interface that all node types must implement
type Node interface {
	GetID() string
	Type() string
	Validate() error
	MarshalJSON() ([]byte, error)
}

// StartNode represents the entry point of a workflow
type StartNode struct {
	ID string `json:"id" yaml:"id"`
}

// GetID returns the node ID
func (n *StartNode) GetID() string {
	return n.ID
}

// Type returns the node type
func (n *StartNode) Type() string {
	return "start"
}

// Validate checks if the start node is valid
func (n *StartNode) Validate() error {
	if n.ID == "" {
		return errors.New("start node: empty node ID")
	}
	return nil
}

// MarshalJSON implements custom JSON marshaling
func (n *StartNode) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}{
		ID:   n.ID,
		Type: "start",
	})
}

// EndNode represents an exit point of a workflow
type EndNode struct {
	ID          string `json:"id" yaml:"id"`
	ReturnValue string `json:"return_value,omitempty" yaml:"return_value,omitempty"`
}

// GetID returns the node ID
func (n *EndNode) GetID() string {
	return n.ID
}

// Type returns the node type
func (n *EndNode) Type() string {
	return "end"
}

// Validate checks if the end node is valid
func (n *EndNode) Validate() error {
	if n.ID == "" {
		return errors.New("end node: empty node ID")
	}
	return nil
}

// MarshalJSON implements custom JSON marshaling
func (n *EndNode) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID          string `json:"id"`
		Type        string `json:"type"`
		ReturnValue string `json:"return_value,omitempty"`
	}{
		ID:          n.ID,
		Type:        "end",
		ReturnValue: n.ReturnValue,
	})
}

// MCPToolNode represents a node that executes an MCP tool
type MCPToolNode struct {
	ID             string            `json:"id" yaml:"id"`
	ServerID       string            `json:"server_id" yaml:"server_id"`
	ToolName       string            `json:"tool_name" yaml:"tool_name"`
	Parameters     map[string]string `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	OutputVariable string            `json:"output_variable" yaml:"output_variable"`
}

// GetID returns the node ID
func (n *MCPToolNode) GetID() string {
	return n.ID
}

// Type returns the node type
func (n *MCPToolNode) Type() string {
	return "mcp_tool"
}

// Validate checks if the MCP tool node is valid
func (n *MCPToolNode) Validate() error {
	if n.ID == "" {
		return errors.New("mcp_tool node: empty node ID")
	}
	if n.ServerID == "" {
		return errors.New("mcp_tool node: empty server ID")
	}
	if n.ToolName == "" {
		return errors.New("mcp_tool node: empty tool name")
	}
	if n.OutputVariable == "" {
		return errors.New("mcp_tool node: empty output variable")
	}
	return nil
}

// MarshalJSON implements custom JSON marshaling
func (n *MCPToolNode) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID             string            `json:"id"`
		Type           string            `json:"type"`
		ServerID       string            `json:"server_id"`
		ToolName       string            `json:"tool_name"`
		Parameters     map[string]string `json:"parameters,omitempty"`
		OutputVariable string            `json:"output_variable"`
	}{
		ID:             n.ID,
		Type:           "mcp_tool",
		ServerID:       n.ServerID,
		ToolName:       n.ToolName,
		Parameters:     n.Parameters,
		OutputVariable: n.OutputVariable,
	})
}

// TransformNode represents a node that transforms data
type TransformNode struct {
	ID             string `json:"id" yaml:"id"`
	InputVariable  string `json:"input_variable" yaml:"input_variable"`
	Expression     string `json:"expression" yaml:"expression"`
	OutputVariable string `json:"output_variable" yaml:"output_variable"`
}

// GetID returns the node ID
func (n *TransformNode) GetID() string {
	return n.ID
}

// Type returns the node type
func (n *TransformNode) Type() string {
	return "transform"
}

// Validate checks if the transform node is valid
func (n *TransformNode) Validate() error {
	if n.ID == "" {
		return errors.New("transform node: empty node ID")
	}
	if n.InputVariable == "" {
		return errors.New("transform node: empty input variable")
	}
	if n.Expression == "" {
		return errors.New("transform node: empty expression")
	}
	if n.OutputVariable == "" {
		return errors.New("transform node: empty output variable")
	}
	return nil
}

// MarshalJSON implements custom JSON marshaling
func (n *TransformNode) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID             string `json:"id"`
		Type           string `json:"type"`
		InputVariable  string `json:"input_variable"`
		Expression     string `json:"expression"`
		OutputVariable string `json:"output_variable"`
	}{
		ID:             n.ID,
		Type:           "transform",
		InputVariable:  n.InputVariable,
		Expression:     n.Expression,
		OutputVariable: n.OutputVariable,
	})
}

// ConditionNode represents a branching node based on a condition
type ConditionNode struct {
	ID        string `json:"id" yaml:"id"`
	Condition string `json:"condition" yaml:"condition"`
}

// GetID returns the node ID
func (n *ConditionNode) GetID() string {
	return n.ID
}

// Type returns the node type
func (n *ConditionNode) Type() string {
	return "condition"
}

// Validate checks if the condition node is valid
func (n *ConditionNode) Validate() error {
	if n.ID == "" {
		return errors.New("condition node: empty node ID")
	}
	if n.Condition == "" {
		return errors.New("condition node: empty condition")
	}
	return nil
}

// MarshalJSON implements custom JSON marshaling
func (n *ConditionNode) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID        string `json:"id"`
		Type      string `json:"type"`
		Condition string `json:"condition"`
	}{
		ID:        n.ID,
		Type:      "condition",
		Condition: n.Condition,
	})
}

// ParallelNode represents a node that executes multiple branches concurrently
type ParallelNode struct {
	ID            string     `json:"id" yaml:"id"`
	Branches      [][]string `json:"branches" yaml:"branches"`
	MergeStrategy string     `json:"merge_strategy" yaml:"merge_strategy"`
}

// GetID returns the node ID
func (n *ParallelNode) GetID() string {
	return n.ID
}

// Type returns the node type
func (n *ParallelNode) Type() string {
	return "parallel"
}

// Validate checks if the parallel node is valid
func (n *ParallelNode) Validate() error {
	if n.ID == "" {
		return errors.New("parallel node: empty node ID")
	}
	if len(n.Branches) == 0 {
		return errors.New("parallel node: empty branches")
	}
	if len(n.Branches) < 2 {
		return errors.New("parallel node: must have at least 2 branches")
	}
	if n.MergeStrategy != "" && n.MergeStrategy != "wait_all" && n.MergeStrategy != "wait_any" && n.MergeStrategy != "wait_first" {
		return fmt.Errorf("parallel node: invalid merge strategy: %s", n.MergeStrategy)
	}
	return nil
}

// MarshalJSON implements custom JSON marshaling
func (n *ParallelNode) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID            string     `json:"id"`
		Type          string     `json:"type"`
		Branches      [][]string `json:"branches"`
		MergeStrategy string     `json:"merge_strategy"`
	}{
		ID:            n.ID,
		Type:          "parallel",
		Branches:      n.Branches,
		MergeStrategy: n.MergeStrategy,
	})
}

// LoopNode represents a node that iterates over a collection
type LoopNode struct {
	ID             string   `json:"id" yaml:"id"`
	Collection     string   `json:"collection" yaml:"collection"`
	ItemVariable   string   `json:"item_variable" yaml:"item_variable"`
	Body           []string `json:"body" yaml:"body"`
	BreakCondition string   `json:"break_condition,omitempty" yaml:"break_condition,omitempty"`
}

// GetID returns the node ID
func (n *LoopNode) GetID() string {
	return n.ID
}

// Type returns the node type
func (n *LoopNode) Type() string {
	return "loop"
}

// Validate checks if the loop node is valid
func (n *LoopNode) Validate() error {
	if n.ID == "" {
		return errors.New("loop node: empty node ID")
	}
	if n.Collection == "" {
		return errors.New("loop node: empty collection")
	}
	if n.ItemVariable == "" {
		return errors.New("loop node: empty item variable")
	}
	if len(n.Body) == 0 {
		return errors.New("loop node: empty body")
	}
	return nil
}

// MarshalJSON implements custom JSON marshaling
func (n *LoopNode) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID             string   `json:"id"`
		Type           string   `json:"type"`
		Collection     string   `json:"collection"`
		ItemVariable   string   `json:"item_variable"`
		Body           []string `json:"body"`
		BreakCondition string   `json:"break_condition,omitempty"`
	}{
		ID:             n.ID,
		Type:           "loop",
		Collection:     n.Collection,
		ItemVariable:   n.ItemVariable,
		Body:           n.Body,
		BreakCondition: n.BreakCondition,
	})
}

// UnmarshalNode unmarshals a JSON node into the appropriate concrete type
func UnmarshalNode(data []byte) (Node, error) {
	// First unmarshal to get the type
	var temp struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return nil, err
	}

	// Unmarshal into the appropriate concrete type
	switch temp.Type {
	case "start":
		var node StartNode
		if err := json.Unmarshal(data, &node); err != nil {
			return nil, err
		}
		return &node, nil
	case "end":
		var node EndNode
		if err := json.Unmarshal(data, &node); err != nil {
			return nil, err
		}
		return &node, nil
	case "mcp_tool":
		var node MCPToolNode
		if err := json.Unmarshal(data, &node); err != nil {
			return nil, err
		}
		return &node, nil
	case "transform":
		var node TransformNode
		if err := json.Unmarshal(data, &node); err != nil {
			return nil, err
		}
		return &node, nil
	case "condition":
		var node ConditionNode
		if err := json.Unmarshal(data, &node); err != nil {
			return nil, err
		}
		return &node, nil
	case "parallel":
		var node ParallelNode
		if err := json.Unmarshal(data, &node); err != nil {
			return nil, err
		}
		return &node, nil
	case "loop":
		var node LoopNode
		if err := json.Unmarshal(data, &node); err != nil {
			return nil, err
		}
		return &node, nil
	default:
		return nil, fmt.Errorf("unknown node type: %s", temp.Type)
	}
}
