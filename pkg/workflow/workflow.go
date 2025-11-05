package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// WorkflowMetadata contains descriptive information about a workflow
type WorkflowMetadata struct {
	Author       string    `json:"author,omitempty" yaml:"author,omitempty"`
	Created      time.Time `json:"created,omitempty" yaml:"created,omitempty"`
	LastModified time.Time `json:"last_modified,omitempty" yaml:"last_modified,omitempty"`
	Tags         []string  `json:"tags,omitempty" yaml:"tags,omitempty"`
	Icon         string    `json:"icon,omitempty" yaml:"icon,omitempty"`
}

// Workflow represents a directed acyclic graph (DAG) of nodes and edges defining an automation workflow
type Workflow struct {
	ID            string           `json:"id" yaml:"id"`
	Name          string           `json:"name" yaml:"name"`
	Version       string           `json:"version" yaml:"version"`
	Description   string           `json:"description,omitempty" yaml:"description,omitempty"`
	Metadata      WorkflowMetadata `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Variables     []*Variable      `json:"variables,omitempty" yaml:"variables,omitempty"`
	ServerConfigs []*ServerConfig  `json:"servers,omitempty" yaml:"servers,omitempty"`
	Nodes         []Node           `json:"nodes,omitempty" yaml:"nodes,omitempty"`
	Edges         []*Edge          `json:"edges,omitempty" yaml:"edges,omitempty"`
}

// NewWorkflow creates a new workflow with the given name and description
func NewWorkflow(name, description string) (*Workflow, error) {
	if name == "" {
		return nil, errors.New("workflow name cannot be empty")
	}

	return &Workflow{
		ID:          NewWorkflowID().String(),
		Name:        name,
		Version:     "1.0.0",
		Description: description,
		Metadata: WorkflowMetadata{
			Created:      time.Now(),
			LastModified: time.Now(),
		},
		Variables:     make([]*Variable, 0),
		ServerConfigs: make([]*ServerConfig, 0),
		Nodes:         make([]Node, 0),
		Edges:         make([]*Edge, 0),
	}, nil
}

// AddNode adds a node to the workflow
func (w *Workflow) AddNode(node Node) error {
	if node == nil {
		return errors.New("cannot add nil node")
	}

	// Check for duplicate node IDs
	nodeID := node.GetID()
	for _, existing := range w.Nodes {
		if existing.GetID() == nodeID {
			return fmt.Errorf("duplicate node ID: %s", nodeID)
		}
	}

	w.Nodes = append(w.Nodes, node)
	w.Metadata.LastModified = time.Now()
	return nil
}

// RemoveNode removes a node from the workflow and all edges connected to it
func (w *Workflow) RemoveNode(nodeID string) error {
	// Find and remove the node
	found := false
	newNodes := make([]Node, 0, len(w.Nodes))
	for _, node := range w.Nodes {
		if node.GetID() != nodeID {
			newNodes = append(newNodes, node)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("node not found: %s", nodeID)
	}

	w.Nodes = newNodes

	// Remove all edges connected to this node
	newEdges := make([]*Edge, 0, len(w.Edges))
	for _, edge := range w.Edges {
		if edge.FromNodeID != nodeID && edge.ToNodeID != nodeID {
			newEdges = append(newEdges, edge)
		}
	}
	w.Edges = newEdges

	w.Metadata.LastModified = time.Now()
	return nil
}

// AddEdge adds an edge to the workflow
// Note: Edges are not validated during addition to allow workflow construction.
// Call Validate() to check all invariants including edge validity.
func (w *Workflow) AddEdge(edge *Edge) error {
	if edge == nil {
		return errors.New("cannot add nil edge")
	}

	// Check for duplicate edges (same from/to pair)
	for _, existing := range w.Edges {
		if existing.FromNodeID == edge.FromNodeID && existing.ToNodeID == edge.ToNodeID {
			return fmt.Errorf("duplicate edge from %s to %s", edge.FromNodeID, edge.ToNodeID)
		}
	}

	// Generate ID if not provided
	if edge.ID == "" {
		edge.ID = NewEdgeID().String()
	}

	w.Edges = append(w.Edges, edge)
	w.Metadata.LastModified = time.Now()
	return nil
}

// RemoveEdge removes an edge from the workflow
func (w *Workflow) RemoveEdge(edgeID string) error {
	found := false
	newEdges := make([]*Edge, 0, len(w.Edges))
	for _, edge := range w.Edges {
		if edge.ID != edgeID {
			newEdges = append(newEdges, edge)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("edge not found: %s", edgeID)
	}

	w.Edges = newEdges
	w.Metadata.LastModified = time.Now()
	return nil
}

// AddVariable adds a variable to the workflow
func (w *Workflow) AddVariable(variable *Variable) error {
	if variable == nil {
		return errors.New("cannot add nil variable")
	}

	// Check for duplicate variable names
	for _, existing := range w.Variables {
		if existing.Name == variable.Name {
			return fmt.Errorf("duplicate variable name: %s", variable.Name)
		}
	}

	w.Variables = append(w.Variables, variable)
	w.Metadata.LastModified = time.Now()
	return nil
}

// GetVariable retrieves a variable by name
func (w *Workflow) GetVariable(name string) (*Variable, error) {
	if name == "" {
		return nil, errors.New("variable name cannot be empty")
	}

	for _, variable := range w.Variables {
		if variable.Name == name {
			return variable, nil
		}
	}

	return nil, fmt.Errorf("variable not found: %s", name)
}

// RemoveVariable removes a variable from the workflow
func (w *Workflow) RemoveVariable(name string) error {
	found := false
	newVariables := make([]*Variable, 0, len(w.Variables))
	for _, variable := range w.Variables {
		if variable.Name != name {
			newVariables = append(newVariables, variable)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("variable not found: %s", name)
	}

	w.Variables = newVariables
	w.Metadata.LastModified = time.Now()
	return nil
}

// UpdateVariable updates an existing variable
func (w *Workflow) UpdateVariable(name string, updated *Variable) error {
	if updated == nil {
		return errors.New("cannot update with nil variable")
	}

	for i, variable := range w.Variables {
		if variable.Name == name {
			w.Variables[i] = updated
			w.Metadata.LastModified = time.Now()
			return nil
		}
	}

	return fmt.Errorf("variable not found: %s", name)
}

// Validate checks all workflow invariants
func (w *Workflow) Validate() error {
	// Invariant 1: Must have exactly one Start node
	startCount := 0
	for _, node := range w.Nodes {
		if node.Type() == "start" {
			startCount++
		}
	}
	if startCount == 0 {
		return errors.New("workflow must have exactly one start node (found 0)")
	}
	if startCount > 1 {
		return fmt.Errorf("workflow must have exactly one start node (found %d)", startCount)
	}

	// Invariant 2: Must have at least one End node
	endCount := 0
	for _, node := range w.Nodes {
		if node.Type() == "end" {
			endCount++
		}
	}
	if endCount == 0 {
		return errors.New("workflow must have at least one end node")
	}

	// Invariant 4: All node IDs must be unique (checked during AddNode)
	nodeIDs := make(map[string]bool)
	for _, node := range w.Nodes {
		nodeID := node.GetID()
		if nodeID == "" {
			return errors.New("found node with empty node ID")
		}
		if nodeIDs[nodeID] {
			return fmt.Errorf("duplicate node ID found: %s", nodeID)
		}
		nodeIDs[nodeID] = true
	}

	// Invariant 5: All variable names must be unique
	variableNames := make(map[string]bool)
	for _, variable := range w.Variables {
		if variable.Name == "" {
			return errors.New("found variable with empty variable name")
		}
		if variableNames[variable.Name] {
			return fmt.Errorf("duplicate variable name found: %s", variable.Name)
		}
		variableNames[variable.Name] = true
	}

	// Validate all variables (skip variables without type - under construction)
	for _, variable := range w.Variables {
		// Skip validation for variables without type (workflow under construction)
		if variable.Type == "" {
			continue
		}
		if err := variable.Validate(); err != nil {
			return fmt.Errorf("variable validation failed: %w", err)
		}
	}

	// Invariant 6: All edges must reference valid node IDs
	for _, edge := range w.Edges {
		if !nodeIDs[edge.FromNodeID] {
			return fmt.Errorf("edge references invalid node reference (from): %s", edge.FromNodeID)
		}
		if !nodeIDs[edge.ToNodeID] {
			return fmt.Errorf("edge references invalid node reference (to): %s", edge.ToNodeID)
		}
	}

	// Validate all edges
	for _, edge := range w.Edges {
		if err := edge.Validate(); err != nil {
			return fmt.Errorf("edge validation failed: %w", err)
		}
	}

	// Validate condition nodes have exactly 2 outgoing edges with conditions
	for _, node := range w.Nodes {
		if node.Type() == "condition" {
			nodeID := node.GetID()
			outgoingEdges := 0
			conditionedEdges := 0
			for _, edge := range w.Edges {
				if edge.FromNodeID == nodeID {
					outgoingEdges++
					if edge.Condition != "" {
						conditionedEdges++
					}
				}
			}
			if outgoingEdges != 2 {
				return fmt.Errorf("condition node %s must have exactly 2 outgoing edges (found %d)", nodeID, outgoingEdges)
			}
			if conditionedEdges != 2 {
				return fmt.Errorf("edges from condition node %s must have conditions", nodeID)
			}
		}
	}

	// Invariant 3: No circular dependencies (DAG property)
	if err := w.checkForCycles(); err != nil {
		return err
	}

	// Invariant 7: No orphaned nodes (all nodes reachable from Start)
	if err := w.checkForOrphanedNodes(); err != nil {
		return err
	}

	return nil
}

// checkForCycles performs depth-first search to detect cycles
func (w *Workflow) checkForCycles() error {
	// If no edges, no cycles possible
	if len(w.Edges) == 0 {
		return nil
	}

	// Build adjacency list
	adjacency := make(map[string][]string)
	for _, edge := range w.Edges {
		adjacency[edge.FromNodeID] = append(adjacency[edge.FromNodeID], edge.ToNodeID)
	}

	// Track visit states: 0=unvisited, 1=visiting, 2=visited
	state := make(map[string]int)

	// DFS function to detect cycles
	var dfs func(string) bool
	dfs = func(nodeID string) bool {
		if state[nodeID] == 1 {
			// Currently visiting this node - found a cycle
			return true
		}
		if state[nodeID] == 2 {
			// Already visited and no cycle found through this path
			return false
		}

		// Mark as currently visiting
		state[nodeID] = 1

		// Visit all neighbors
		for _, neighbor := range adjacency[nodeID] {
			if dfs(neighbor) {
				return true
			}
		}

		// Mark as fully visited
		state[nodeID] = 2
		return false
	}

	// Check from each node (to handle disconnected components)
	for _, node := range w.Nodes {
		nodeID := node.GetID()
		if state[nodeID] == 0 {
			if dfs(nodeID) {
				return errors.New("workflow contains circular dependency")
			}
		}
	}

	return nil
}

// checkForOrphanedNodes checks that all nodes are reachable from Start node
func (w *Workflow) checkForOrphanedNodes() error {
	// If there are no edges, skip this check (workflow under construction)
	if len(w.Edges) == 0 {
		return nil
	}

	// Find start node
	var startNodeID string
	for _, node := range w.Nodes {
		if node.Type() == "start" {
			startNodeID = node.GetID()
			break
		}
	}

	if startNodeID == "" {
		// Already checked in Validate, but be defensive
		return errors.New("no start node found")
	}

	// Build adjacency list (both forward and backward edges for reachability)
	adjacency := make(map[string][]string)
	for _, edge := range w.Edges {
		adjacency[edge.FromNodeID] = append(adjacency[edge.FromNodeID], edge.ToNodeID)
	}

	// BFS from start node to find all reachable nodes
	reachable := make(map[string]bool)
	queue := []string{startNodeID}
	reachable[startNodeID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, neighbor := range adjacency[current] {
			if !reachable[neighbor] {
				reachable[neighbor] = true
				queue = append(queue, neighbor)
			}
		}
	}

	// Check if all nodes are reachable
	for _, node := range w.Nodes {
		nodeID := node.GetID()
		if !reachable[nodeID] {
			return fmt.Errorf("orphaned node (not reachable from start): %s", nodeID)
		}
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling for Workflow
func (w *Workflow) MarshalJSON() ([]byte, error) {
	type Alias Workflow
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(w),
	})
}
