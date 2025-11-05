package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Edge represents a connection between two nodes in a workflow
type Edge struct {
	ID         string `json:"id" yaml:"id,omitempty"`
	FromNodeID string `json:"from_node_id" yaml:"from,omitempty"`
	ToNodeID   string `json:"to_node_id" yaml:"to,omitempty"`
	Condition  string `json:"condition,omitempty" yaml:"condition,omitempty"`
	Label      string `json:"label,omitempty" yaml:"label,omitempty"`
}

// Validate checks if the edge is valid
func (e *Edge) Validate() error {
	if e.ID == "" {
		return errors.New("edge: empty edge ID")
	}
	if e.FromNodeID == "" {
		return errors.New("edge: empty from node")
	}
	if e.ToNodeID == "" {
		return errors.New("edge: empty to node")
	}
	if e.FromNodeID == e.ToNodeID {
		return fmt.Errorf("edge: self-loop detected (node %s to itself)", e.FromNodeID)
	}
	return nil
}

// MarshalJSON implements custom JSON marshaling for Edge
func (e *Edge) MarshalJSON() ([]byte, error) {
	type Alias Edge
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(e),
	})
}

// UnmarshalJSON implements custom JSON unmarshaling for Edge
func (e *Edge) UnmarshalJSON(data []byte) error {
	type Alias Edge
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(e),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	return nil
}
