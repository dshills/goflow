package tui

import (
	"fmt"
	"strings"

	"github.com/dshills/goflow/pkg/workflow"
)

// ValidateWorkflow performs full workflow validation
// Returns ValidationStatus with all errors and warnings
// Complexity: O(V + E) where V = nodes, E = edges
func ValidateWorkflow(wf *workflow.Workflow) *ValidationStatus {
	status := NewValidationStatus()

	if wf == nil {
		status.AddError("", "nil_workflow", "Workflow is nil")
		status.SetValidated()
		return status
	}

	// Check for circular dependencies (O(V + E))
	_ = checkCircularDependencies(wf, status) // Errors added to status

	// Check all nodes reachable from start (O(V + E))
	_ = checkReachability(wf, status) // Errors added to status

	// Validate each node (O(V))
	for _, node := range wf.Nodes {
		if node == nil {
			status.AddError("", "nil_node", "Workflow contains nil node")
			continue
		}

		nodeErrors := ValidateNode(node, wf)
		for _, err := range nodeErrors {
			status.AddError(err.NodeID, err.ErrorType, err.Message)
		}
	}

	// Validate each edge (O(E))
	nodeIDSet := buildNodeIDSet(wf)
	for _, edge := range wf.Edges {
		if edge == nil {
			status.AddError("", "nil_edge", "Workflow contains nil edge")
			continue
		}

		if !nodeIDSet[edge.FromNodeID] {
			status.AddError(
				edge.FromNodeID,
				"invalid_edge_target",
				fmt.Sprintf("Edge from non-existent node '%s'", edge.FromNodeID),
			)
		}

		if !nodeIDSet[edge.ToNodeID] {
			status.AddError(
				edge.FromNodeID,
				"invalid_edge_target",
				fmt.Sprintf("Edge from '%s' targets non-existent node '%s'", edge.FromNodeID, edge.ToNodeID),
			)
		}
	}

	// Check domain-specific rules (O(V + E))
	checkDomainRules(wf, status)

	status.SetValidated()
	return status
}

// ValidateNode performs single node validation
// Returns list of validation errors for the node
func ValidateNode(node workflow.Node, wf *workflow.Workflow) []ValidationError {
	var errors []ValidationError

	if node == nil {
		return []ValidationError{{
			NodeID:    "",
			ErrorType: "nil_node",
			Message:   "Node is nil",
		}}
	}

	nodeID := node.GetID()

	// Validate required fields based on node type
	switch n := node.(type) {
	case *workflow.StartNode:
		if n.ID == "" {
			errors = append(errors, ValidationError{
				NodeID:    nodeID,
				ErrorType: "missing_required_field",
				Message:   "Required field 'id' missing in start node",
			})
		}

	case *workflow.EndNode:
		if n.ID == "" {
			errors = append(errors, ValidationError{
				NodeID:    nodeID,
				ErrorType: "missing_required_field",
				Message:   "Required field 'id' missing in end node",
			})
		}

	case *workflow.MCPToolNode:
		if n.ServerID == "" {
			errors = append(errors, ValidationError{
				NodeID:    nodeID,
				ErrorType: "missing_required_field",
				Message:   "Required field 'server_id' missing in MCP tool node",
			})
		}
		if n.ToolName == "" {
			errors = append(errors, ValidationError{
				NodeID:    nodeID,
				ErrorType: "missing_required_field",
				Message:   "Required field 'tool_name' missing in MCP tool node",
			})
		}
		if n.OutputVariable == "" {
			errors = append(errors, ValidationError{
				NodeID:    nodeID,
				ErrorType: "missing_required_field",
				Message:   "Required field 'output_variable' missing in MCP tool node",
			})
		}

		// Validate template syntax in parameters
		for key, value := range n.Parameters {
			if containsTemplate(value) {
				if err := workflow.ValidateTemplateSyntax(value); err != nil {
					errors = append(errors, ValidationError{
						NodeID:    nodeID,
						ErrorType: "invalid_template",
						Message:   fmt.Sprintf("Invalid template in parameter '%s': %v", key, err),
					})
				}
			}
		}

	case *workflow.TransformNode:
		if n.InputVariable == "" {
			errors = append(errors, ValidationError{
				NodeID:    nodeID,
				ErrorType: "missing_required_field",
				Message:   "Required field 'input_variable' missing in transform node",
			})
		}
		if n.Expression == "" {
			errors = append(errors, ValidationError{
				NodeID:    nodeID,
				ErrorType: "missing_required_field",
				Message:   "Required field 'expression' missing in transform node",
			})
		}
		if n.OutputVariable == "" {
			errors = append(errors, ValidationError{
				NodeID:    nodeID,
				ErrorType: "missing_required_field",
				Message:   "Required field 'output_variable' missing in transform node",
			})
		}

		// Validate expression syntax
		if n.Expression != "" {
			// Check if it's a JSONPath expression
			if len(n.Expression) > 0 && n.Expression[0] == '$' {
				if err := workflow.ValidateJSONPathSyntax(n.Expression); err != nil {
					errors = append(errors, ValidationError{
						NodeID:    nodeID,
						ErrorType: "invalid_jsonpath",
						Message:   fmt.Sprintf("Invalid JSONPath: %v", err),
					})
				}
			}

			// Check if it contains templates
			if containsTemplate(n.Expression) {
				if err := workflow.ValidateTemplateSyntax(n.Expression); err != nil {
					errors = append(errors, ValidationError{
						NodeID:    nodeID,
						ErrorType: "invalid_template",
						Message:   fmt.Sprintf("Invalid template syntax: %v", err),
					})
				}
			}
		}

	case *workflow.ConditionNode:
		if n.Condition == "" {
			errors = append(errors, ValidationError{
				NodeID:    nodeID,
				ErrorType: "missing_required_field",
				Message:   "Required field 'condition' missing in condition node",
			})
		}

		// Validate expression syntax
		if n.Condition != "" {
			if err := workflow.ValidateExpressionSyntax(n.Condition); err != nil {
				errors = append(errors, ValidationError{
					NodeID:    nodeID,
					ErrorType: "invalid_expression",
					Message:   fmt.Sprintf("Invalid expression syntax: %v", err),
				})
			}
		}

	case *workflow.LoopNode:
		if n.Collection == "" {
			errors = append(errors, ValidationError{
				NodeID:    nodeID,
				ErrorType: "missing_required_field",
				Message:   "Required field 'collection' missing in loop node",
			})
		}
		if n.ItemVariable == "" {
			errors = append(errors, ValidationError{
				NodeID:    nodeID,
				ErrorType: "missing_required_field",
				Message:   "Required field 'item_variable' missing in loop node",
			})
		}
		if len(n.Body) == 0 {
			errors = append(errors, ValidationError{
				NodeID:    nodeID,
				ErrorType: "missing_required_field",
				Message:   "Loop node must have at least one body node",
			})
		}

	case *workflow.ParallelNode:
		if len(n.Branches) == 0 {
			errors = append(errors, ValidationError{
				NodeID:    nodeID,
				ErrorType: "missing_required_field",
				Message:   "Parallel node must have at least one branch",
			})
		}
	}

	return errors
}

// checkCircularDependencies detects cycles in the workflow graph using DFS
func checkCircularDependencies(wf *workflow.Workflow, status *ValidationStatus) error {
	// Build adjacency list
	adjacency := make(map[string][]string)
	for _, edge := range wf.Edges {
		adjacency[edge.FromNodeID] = append(adjacency[edge.FromNodeID], edge.ToNodeID)
	}

	// Track visit states: 0=unvisited, 1=visiting, 2=visited
	state := make(map[string]int)
	parent := make(map[string]string) // Track parent for cycle path reconstruction

	// DFS function to detect cycles
	var dfs func(string, string) []string
	dfs = func(nodeID, parentID string) []string {
		if state[nodeID] == 1 {
			// Currently visiting this node - found a cycle
			// Reconstruct the cycle path
			cycle := []string{nodeID}
			current := parentID
			for current != nodeID && current != "" {
				cycle = append([]string{current}, cycle...)
				current = parent[current]
			}
			cycle = append([]string{nodeID}, cycle...)
			return cycle
		}
		if state[nodeID] == 2 {
			// Already visited and no cycle found through this path
			return nil
		}

		// Mark as currently visiting
		state[nodeID] = 1
		parent[nodeID] = parentID

		// Visit all neighbors
		for _, neighbor := range adjacency[nodeID] {
			if cycle := dfs(neighbor, nodeID); cycle != nil {
				return cycle
			}
		}

		// Mark as fully visited
		state[nodeID] = 2
		return nil
	}

	// Check from each node (to handle disconnected components)
	for _, node := range wf.Nodes {
		nodeID := node.GetID()
		if state[nodeID] == 0 {
			if cycle := dfs(nodeID, ""); cycle != nil {
				cyclePath := strings.Join(cycle, " → ")
				status.AddError("", "circular_dependency", fmt.Sprintf("Circular dependency detected: %s", cyclePath))
				return fmt.Errorf("circular dependency")
			}
		}
	}

	return nil
}

// checkReachability checks that all nodes are reachable from start using BFS
func checkReachability(wf *workflow.Workflow, status *ValidationStatus) error {
	// If there are no edges, skip this check (workflow under construction)
	if len(wf.Edges) == 0 {
		return nil
	}

	// Find start node
	var startNodeID string
	for _, node := range wf.Nodes {
		if node.Type() == "start" {
			startNodeID = node.GetID()
			break
		}
	}

	if startNodeID == "" {
		status.AddError("", "no_start_node", "Workflow must have a start node")
		return fmt.Errorf("no start node")
	}

	// Build adjacency list
	adjacency := make(map[string][]string)
	for _, edge := range wf.Edges {
		adjacency[edge.FromNodeID] = append(adjacency[edge.FromNodeID], edge.ToNodeID)
	}

	// Add implicit connections from parallel and loop nodes
	for _, node := range wf.Nodes {
		switch n := node.(type) {
		case *workflow.ParallelNode:
			for _, branch := range n.Branches {
				adjacency[n.ID] = append(adjacency[n.ID], branch...)
			}
		case *workflow.LoopNode:
			adjacency[n.ID] = append(adjacency[n.ID], n.Body...)
		}
	}

	// BFS from start node
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

	// Check if all nodes are reachable (warning, not error)
	for _, node := range wf.Nodes {
		nodeID := node.GetID()
		if !reachable[nodeID] {
			status.AddWarning(nodeID, fmt.Sprintf("Node '%s' is not reachable from start", nodeID))
		}
	}

	return nil
}

// checkDomainRules validates domain-specific rules
func checkDomainRules(wf *workflow.Workflow, status *ValidationStatus) {
	// Build edge count map
	outgoingEdges := make(map[string]int)
	for _, edge := range wf.Edges {
		outgoingEdges[edge.FromNodeID]++
	}

	for _, node := range wf.Nodes {
		nodeID := node.GetID()

		switch n := node.(type) {
		case *workflow.ConditionNode:
			// Condition nodes must have exactly 2 outgoing edges
			count := outgoingEdges[nodeID]
			if count != 2 {
				status.AddError(
					nodeID,
					"invalid_condition_edges",
					fmt.Sprintf("Condition node '%s' must have exactly 2 outgoing edges (true/false), found %d", nodeID, count),
				)
			}

		case *workflow.LoopNode:
			// Loop nodes should have valid collection source
			// Check if collection variable exists
			collection := n.Collection
			if collection != "" {
				// Remove template syntax if present
				collection = strings.TrimPrefix(collection, "${")
				collection = strings.TrimSuffix(collection, "}")

				found := false
				for _, v := range wf.Variables {
					if v.Name == collection {
						found = true
						break
					}
				}

				if !found {
					status.AddWarning(
						nodeID,
						fmt.Sprintf("Loop node '%s' references undefined collection variable '%s'", nodeID, collection),
					)
				}
			}

		case *workflow.ParallelNode:
			// Parallel nodes must have at least 2 branches
			if len(n.Branches) < 2 {
				status.AddError(
					nodeID,
					"invalid_parallel_branches",
					fmt.Sprintf("Parallel node '%s' must have at least 2 branches, found %d", nodeID, len(n.Branches)),
				)
			}
		}
	}
}

// buildNodeIDSet creates a set of all node IDs for quick lookup
func buildNodeIDSet(wf *workflow.Workflow) map[string]bool {
	nodeIDs := make(map[string]bool)
	for _, node := range wf.Nodes {
		if node != nil {
			nodeIDs[node.GetID()] = true
		}
	}
	return nodeIDs
}

// containsTemplate checks if a string contains template syntax ${...}
func containsTemplate(s string) bool {
	return strings.Contains(s, "${")
}
