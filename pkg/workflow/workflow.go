package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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
// Note: Nodes are not validated during addition to allow workflow construction.
// Validation is automatically performed before execution, or call Validate() explicitly.
// IMPORTANT: Duplicate node IDs are allowed during construction but will cause
// validation errors. This enables flexible workflow building without premature failures.
func (w *Workflow) AddNode(node Node) error {
	if node == nil {
		return errors.New("cannot add nil node")
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
// Note: Variables are not validated during addition to allow workflow construction.
// Validation is automatically performed before execution, or call Validate() explicitly.
// IMPORTANT: Duplicate variable names are allowed during construction but will cause
// validation errors. This enables flexible workflow building without premature failures.
func (w *Workflow) AddVariable(variable *Variable) error {
	if variable == nil {
		return errors.New("cannot add nil variable")
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
	var validationErrors []string

	// Invariant 1: Must have exactly one Start node
	startCount := 0
	for _, node := range w.Nodes {
		if node.Type() == "start" {
			startCount++
		}
	}
	if startCount == 0 {
		validationErrors = append(validationErrors, "workflow must have exactly one start node (found 0)")
	}
	if startCount > 1 {
		validationErrors = append(validationErrors, fmt.Sprintf("workflow must have exactly one start node (found %d)", startCount))
	}

	// Invariant 2: Must have at least one End node
	endCount := 0
	for _, node := range w.Nodes {
		if node.Type() == "end" {
			endCount++
		}
	}
	if endCount == 0 {
		validationErrors = append(validationErrors, "workflow must have at least one end node")
	}

	// Invariant 4: All node IDs must be unique (checked during AddNode)
	nodeIDs := make(map[string]bool)
	for _, node := range w.Nodes {
		nodeID := node.GetID()
		if nodeID == "" {
			validationErrors = append(validationErrors, "found node with empty node ID")
			continue
		}
		if nodeIDs[nodeID] {
			validationErrors = append(validationErrors, fmt.Sprintf("duplicate node ID found: %s", nodeID))
		}
		nodeIDs[nodeID] = true
	}

	// Invariant 5: All variable names must be unique
	variableNames := make(map[string]bool)
	for _, variable := range w.Variables {
		if variable.Name == "" {
			validationErrors = append(validationErrors, "found variable with empty variable name")
			continue
		}
		if variableNames[variable.Name] {
			validationErrors = append(validationErrors, fmt.Sprintf("duplicate variable name found: %s", variable.Name))
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
			validationErrors = append(validationErrors, fmt.Sprintf("variable validation failed: %v", err))
		}
	}

	// Invariant 6: All edges must reference valid node IDs
	for _, edge := range w.Edges {
		if !nodeIDs[edge.FromNodeID] {
			validationErrors = append(validationErrors, fmt.Sprintf("edge references invalid node reference (from): %s", edge.FromNodeID))
		}
		if !nodeIDs[edge.ToNodeID] {
			validationErrors = append(validationErrors, fmt.Sprintf("edge references invalid node reference (to): %s", edge.ToNodeID))
		}
	}

	// Validate all edges
	for _, edge := range w.Edges {
		if err := edge.Validate(); err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("edge validation failed: %v", err))
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
				validationErrors = append(validationErrors, fmt.Sprintf("condition node %s must have exactly 2 outgoing edges (found %d)", nodeID, outgoingEdges))
			}
			if conditionedEdges != 2 {
				validationErrors = append(validationErrors, fmt.Sprintf("edges from condition node %s must have conditions", nodeID))
			}
		}
	}

	// Validate expressions in nodes
	for _, node := range w.Nodes {
		switch n := node.(type) {
		case *ConditionNode:
			if err := w.validateConditionExpression(n); err != nil {
				validationErrors = append(validationErrors, fmt.Sprintf("node %s: %v", n.GetID(), err))
			}
		case *TransformNode:
			if err := w.validateTransformConfig(n); err != nil {
				validationErrors = append(validationErrors, fmt.Sprintf("node %s: %v", n.GetID(), err))
			}
		case *MCPToolNode:
			if err := w.validateMCPToolNode(n); err != nil {
				validationErrors = append(validationErrors, fmt.Sprintf("node %s: %v", n.GetID(), err))
			}
		}
	}

	// Invariant 3: No circular dependencies (DAG property)
	if err := w.checkForCycles(); err != nil {
		validationErrors = append(validationErrors, err.Error())
	}

	// Invariant 7: No orphaned nodes (all nodes reachable from Start)
	if err := w.checkForOrphanedNodes(); err != nil {
		validationErrors = append(validationErrors, err.Error())
	}

	// Return combined errors if any
	if len(validationErrors) > 0 {
		return errors.New(strings.Join(validationErrors, "; "))
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

	// Add implicit connections from parallel and loop nodes to their branch/body nodes
	for _, node := range w.Nodes {
		switch n := node.(type) {
		case *ParallelNode:
			// Parallel nodes connect to all branch nodes
			for _, branch := range n.Branches {
				adjacency[n.ID] = append(adjacency[n.ID], branch...)
			}
		case *LoopNode:
			// Loop nodes connect to all body nodes
			adjacency[n.ID] = append(adjacency[n.ID], n.Body...)
		}
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

// validateConditionExpression validates the condition expression in a ConditionNode
func (w *Workflow) validateConditionExpression(node *ConditionNode) error {
	if node.Condition == "" {
		return errors.New("condition expression cannot be empty")
	}

	// Try to compile the expression to validate syntax (this also checks for unsafe operations)
	// Note: We use a minimal context for validation - actual values will be provided at runtime
	// This validates syntax without requiring actual data
	if err := validateExpressionSyntax(node.Condition); err != nil {
		return fmt.Errorf("invalid condition expression: %w", err)
	}

	// Extract variable references from the expression
	varRefs := extractVariableReferences(node.Condition)

	// Check that all referenced variables are defined in the workflow
	for _, varName := range varRefs {
		if !w.hasVariable(varName) && !w.hasNodeOutput(varName) && !w.isLoopItemVariable(varName) {
			return fmt.Errorf("undefined variable in condition: %s", varName)
		}
	}

	return nil
}

// validateTransformConfig validates the transformation configuration in a TransformNode
func (w *Workflow) validateTransformConfig(node *TransformNode) error {
	if node.Expression == "" {
		return errors.New("transform expression cannot be empty")
	}

	if node.InputVariable == "" {
		return errors.New("transform input variable cannot be empty")
	}

	if node.OutputVariable == "" {
		return errors.New("transform output variable cannot be empty")
	}

	// Validate that input variable is defined
	// Input can be either a template "${var}" or a plain variable name "var"
	if containsTemplate(node.InputVariable) {
		// Extract variable names from template
		inputVars := extractTemplateVariables(node.InputVariable)
		for _, varName := range inputVars {
			if !w.hasVariable(varName) && !w.hasNodeOutput(varName) && !w.isLoopItemVariable(varName) {
				return fmt.Errorf("undefined input variable: %s", node.InputVariable)
			}
		}
	} else {
		// Plain variable name - validate directly
		if !w.hasVariable(node.InputVariable) && !w.hasNodeOutput(node.InputVariable) && !w.isLoopItemVariable(node.InputVariable) {
			return fmt.Errorf("undefined input variable: %s", node.InputVariable)
		}
	}

	// Validate the expression syntax based on its type
	expr := node.Expression

	// Check if it's a JSONPath expression (starts with $)
	if len(expr) > 0 && expr[0] == '$' {
		if err := validateJSONPathSyntax(expr); err != nil {
			return fmt.Errorf("invalid JSONPath expression: %w", err)
		}
	}

	// Check if it's a template (contains ${...})
	if containsTemplate(expr) {
		if err := validateTemplateSyntax(expr); err != nil {
			return fmt.Errorf("invalid template syntax: %w", err)
		}
		// Extract variables from template and validate
		varRefs := extractTemplateVariables(expr)
		for _, varName := range varRefs {
			// Skip special transform variables like "input" which are provided at runtime
			if varName == "input" {
				continue
			}
			if !w.hasVariable(varName) && !w.hasNodeOutput(varName) && !w.isLoopItemVariable(varName) {
				return fmt.Errorf("undefined variable in template: %s", varName)
			}
		}
	}

	return nil
}

// validateMCPToolNode validates MCP tool node configuration
func (w *Workflow) validateMCPToolNode(node *MCPToolNode) error {
	// Validate server reference
	if node.ServerID != "" {
		serverExists := false
		for _, server := range w.ServerConfigs {
			if server.ID == node.ServerID {
				serverExists = true
				break
			}
		}
		if !serverExists {
			return fmt.Errorf("undefined server: %s", node.ServerID)
		}
	}

	// Validate variables in parameters
	if node.Parameters != nil {
		for key, value := range node.Parameters {
			// Check if it's a template string
			if containsTemplate(value) {
				if err := validateTemplateSyntax(value); err != nil {
					return fmt.Errorf("invalid template syntax in parameter %s: %w", key, err)
				}
				// Extract and validate variables
				varRefs := extractTemplateVariables(value)
				for _, varName := range varRefs {
					if !w.hasVariable(varName) && !w.hasNodeOutput(varName) && !w.isLoopItemVariable(varName) {
						return fmt.Errorf("undefined variable: %s", varName)
					}
				}
			}
		}
	}

	return nil
}

// hasVariable checks if a variable with the given name exists in the workflow
func (w *Workflow) hasVariable(name string) bool {
	for _, v := range w.Variables {
		if v.Name == name {
			return true
		}
	}
	return false
}

// isLoopItemVariable checks if a variable name is a loop item variable
func (w *Workflow) isLoopItemVariable(name string) bool {
	for _, node := range w.Nodes {
		if loopNode, ok := node.(*LoopNode); ok {
			if loopNode.ItemVariable == name {
				return true
			}
		}
	}
	return false
}

// hasNodeOutput checks if a variable is an output from any node
func (w *Workflow) hasNodeOutput(name string) bool {
	for _, node := range w.Nodes {
		switch n := node.(type) {
		case *MCPToolNode:
			if n.OutputVariable == name {
				return true
			}
		case *TransformNode:
			if n.OutputVariable == name {
				return true
			}
		}
	}
	return false
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
