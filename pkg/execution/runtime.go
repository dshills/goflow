package execution

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/domain/types"
	"github.com/dshills/goflow/pkg/mcpserver"
	"github.com/dshills/goflow/pkg/storage"
	"github.com/dshills/goflow/pkg/workflow"
)

// Engine is the workflow execution runtime engine that orchestrates workflow execution.
type Engine struct {
	serverRegistry *mcpserver.Registry
	execRepository *storage.SQLiteExecutionRepository
	logger         *Logger
}

// NewEngine creates a new execution engine with default configuration.
func NewEngine() *Engine {
	// Create execution repository
	repo, err := storage.NewSQLiteExecutionRepository()
	if err != nil {
		// For now, continue without persistence if DB fails
		// In production, this should be handled more gracefully
		repo = nil
	}

	logger := NewLogger(repo)

	return &Engine{
		serverRegistry: mcpserver.NewRegistry(),
		execRepository: repo,
		logger:         logger,
	}
}

// NewEngineWithRepository creates an engine with a custom repository (useful for testing).
func NewEngineWithRepository(repo *storage.SQLiteExecutionRepository) *Engine {
	logger := NewLogger(repo)

	return &Engine{
		serverRegistry: mcpserver.NewRegistry(),
		execRepository: repo,
		logger:         logger,
	}
}

// Execute runs a workflow with the given inputs and returns the execution result.
// This is the main entry point for workflow execution.
func (e *Engine) Execute(ctx context.Context, wf *workflow.Workflow, inputs map[string]interface{}) (*execution.Execution, error) {
	// Validate workflow first
	if err := wf.Validate(); err != nil {
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}

	// Create execution entity
	exec, err := execution.NewExecution(types.WorkflowID(wf.ID), wf.Version, inputs)
	if err != nil {
		return nil, fmt.Errorf("failed to create execution: %w", err)
	}

	// Validate required input variables
	if err := e.validateInputs(wf, inputs); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	// Initialize workflow variables with defaults
	if err := e.initializeVariables(exec.Context, wf); err != nil {
		return nil, fmt.Errorf("failed to initialize variables: %w", err)
	}

	// Log execution start
	if e.logger != nil {
		e.logger.LogExecutionStart(exec)
	}

	// Start execution
	if err := exec.Start(); err != nil {
		return exec, fmt.Errorf("failed to start execution: %w", err)
	}

	// Connect to MCP servers
	if err := e.connectServers(ctx, wf); err != nil {
		execErr := &execution.ExecutionError{
			Type:        execution.ErrorTypeConnection,
			Message:     fmt.Sprintf("failed to connect to MCP servers: %v", err),
			Timestamp:   time.Now(),
			Recoverable: true,
		}
		_ = exec.Fail(execErr)
		if e.logger != nil {
			e.logger.LogExecutionComplete(exec)
		}
		return exec, err
	}

	// Ensure servers are disconnected when done
	defer e.disconnectServers(wf)

	// Execute workflow
	if err := e.executeWorkflow(ctx, wf, exec); err != nil {
		// Check if context was cancelled
		if ctx.Err() == context.Canceled {
			_ = exec.Cancel()
		} else {
			// Execution failed
			execErr, ok := err.(*execution.ExecutionError)
			if !ok {
				// Wrap generic errors
				execErr = &execution.ExecutionError{
					Type:        execution.ErrorTypeExecution,
					Message:     err.Error(),
					Timestamp:   time.Now(),
					StackTrace:  string(debug.Stack()),
					Recoverable: false,
				}
			}
			_ = exec.Fail(execErr)
		}

		if e.logger != nil {
			e.logger.LogExecutionComplete(exec)
		}
		return exec, err
	}

	// Mark execution as completed (return value set by End node)
	if err := exec.Complete(exec.ReturnValue); err != nil {
		return exec, fmt.Errorf("failed to complete execution: %w", err)
	}

	// Log execution completion
	if e.logger != nil {
		e.logger.LogExecutionComplete(exec)
	}

	return exec, nil
}

// executeWorkflow orchestrates the execution of nodes following the workflow graph.
func (e *Engine) executeWorkflow(ctx context.Context, wf *workflow.Workflow, exec *execution.Execution) error {
	// Find the start node
	var startNode workflow.Node
	for _, node := range wf.Nodes {
		if node.Type() == "start" {
			startNode = node
			break
		}
	}

	if startNode == nil {
		return &execution.ExecutionError{
			Type:        execution.ErrorTypeValidation,
			Message:     "no start node found in workflow",
			Timestamp:   time.Now(),
			Recoverable: false,
		}
	}

	// Create a map for quick node lookup
	nodeMap := make(map[string]workflow.Node)
	for _, node := range wf.Nodes {
		nodeMap[node.GetID()] = node
	}

	// Execute workflow starting from start node
	visited := make(map[string]bool)
	return e.executeNodePath(ctx, startNode, wf, exec, nodeMap, visited)
}

// executeNodePath executes a node and follows the appropriate edges based on execution results.
func (e *Engine) executeNodePath(ctx context.Context, node workflow.Node, wf *workflow.Workflow, exec *execution.Execution, nodeMap map[string]workflow.Node, visited map[string]bool) error {
	nodeID := node.GetID()

	// Prevent infinite loops
	if visited[nodeID] {
		return nil
	}
	visited[nodeID] = true

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Execute the current node
	nodeExec, err := e.executeNodeAndGetExecution(ctx, node, wf, exec)
	if err != nil {
		return err
	}

	// If this is an end node, stop here
	if node.Type() == "end" {
		return nil
	}

	// Get next nodes to execute based on edges
	nextNodes, err := e.getNextNodes(nodeID, wf, nodeExec)
	if err != nil {
		return &execution.ExecutionError{
			Type:        execution.ErrorTypeExecution,
			Message:     fmt.Sprintf("failed to determine next nodes from %s: %v", nodeID, err),
			NodeID:      types.NodeID(nodeID),
			Timestamp:   time.Now(),
			Recoverable: false,
		}
	}

	// Execute next nodes
	for _, nextNodeID := range nextNodes {
		nextNode, exists := nodeMap[nextNodeID]
		if !exists {
			return &execution.ExecutionError{
				Type:        execution.ErrorTypeValidation,
				Message:     fmt.Sprintf("next node %s not found", nextNodeID),
				Timestamp:   time.Now(),
				Recoverable: false,
			}
		}

		if err := e.executeNodePath(ctx, nextNode, wf, exec, nodeMap, visited); err != nil {
			return err
		}
	}

	return nil
}

// executeNodeAndGetExecution executes a node and returns its execution record.
func (e *Engine) executeNodeAndGetExecution(ctx context.Context, node workflow.Node, wf *workflow.Workflow, exec *execution.Execution) (*execution.NodeExecution, error) {
	if err := e.executeNode(ctx, node, wf, exec); err != nil {
		return nil, err
	}

	// Find the most recent execution for this node
	nodeID := types.NodeID(node.GetID())
	for i := len(exec.NodeExecutions) - 1; i >= 0; i-- {
		if exec.NodeExecutions[i].NodeID == nodeID {
			return exec.NodeExecutions[i], nil
		}
	}

	return nil, fmt.Errorf("node execution not found for node %s", nodeID)
}

// getNextNodes determines which nodes to execute next based on edges and condition results.
func (e *Engine) getNextNodes(currentNodeID string, wf *workflow.Workflow, nodeExec *execution.NodeExecution) ([]string, error) {
	// Get all edges from current node
	var edges []*workflow.Edge
	for _, edge := range wf.Edges {
		if edge.FromNodeID == currentNodeID {
			edges = append(edges, edge)
		}
	}

	if len(edges) == 0 {
		return nil, nil
	}

	// If this is a condition node, select edge based on boolean result
	if nodeExec != nil && nodeExec.NodeType == "condition" {
		// Get the condition result from outputs
		result, ok := nodeExec.Outputs["result"]
		if !ok {
			return nil, fmt.Errorf("condition node %s did not produce a result", currentNodeID)
		}

		boolResult, ok := result.(bool)
		if !ok {
			return nil, fmt.Errorf("condition node %s result is not boolean: %T", currentNodeID, result)
		}

		// Find the edge matching the condition result
		var matchedEdge *workflow.Edge
		for _, edge := range edges {
			if edge.Condition == "" {
				continue
			}
			if (edge.Condition == "true" && boolResult) || (edge.Condition == "false" && !boolResult) {
				matchedEdge = edge
				break
			}
		}

		if matchedEdge == nil {
			return nil, fmt.Errorf("no edge found for condition result: %v from node %s", boolResult, currentNodeID)
		}

		return []string{matchedEdge.ToNodeID}, nil
	}

	// For non-condition nodes, follow all outgoing edges
	var nextNodes []string
	for _, edge := range edges {
		nextNodes = append(nextNodes, edge.ToNodeID)
	}

	return nextNodes, nil
}

// executeNode executes a single node based on its type.
func (e *Engine) executeNode(ctx context.Context, node workflow.Node, wf *workflow.Workflow, exec *execution.Execution) error {
	nodeID := types.NodeID(node.GetID())

	// Create node execution record
	nodeExec := execution.NewNodeExecution(exec.ID, nodeID, node.Type())
	nodeExec.Start()

	// Set current node in context
	exec.Context.SetCurrentNode(&nodeID)
	defer exec.Context.SetCurrentNode(nil)

	// Execute based on node type
	var err error
	switch n := node.(type) {
	case *workflow.StartNode:
		err = e.executeStartNode(ctx, n, exec, nodeExec)
	case *workflow.EndNode:
		err = e.executeEndNode(ctx, n, exec, nodeExec)
	case *workflow.MCPToolNode:
		err = e.executeMCPToolNode(ctx, n, wf, exec, nodeExec)
	case *workflow.TransformNode:
		err = e.executeTransformNode(ctx, n, exec, nodeExec)
	case *workflow.ConditionNode:
		err = e.executeConditionNode(ctx, n, exec, nodeExec)
	case *workflow.PassthroughNode:
		// Passthrough nodes do nothing, just complete successfully
		nodeExec.Complete(nil)
	default:
		err = fmt.Errorf("unsupported node type: %s", node.Type())
	}

	// Handle node execution result
	if err != nil {
		nodeErr := &execution.NodeError{
			Type:       execution.ErrorTypeExecution,
			Message:    err.Error(),
			StackTrace: string(debug.Stack()),
		}
		nodeExec.Fail(nodeErr)

		// Add to execution record
		_ = exec.AddNodeExecution(nodeExec)

		// Log node execution
		if e.logger != nil {
			e.logger.LogNodeExecution(nodeExec)
		}

		// Return as execution error
		return &execution.ExecutionError{
			Type:        execution.ErrorTypeExecution,
			Message:     fmt.Sprintf("node %s failed: %v", nodeID, err),
			NodeID:      nodeID,
			Timestamp:   time.Now(),
			StackTrace:  string(debug.Stack()),
			Recoverable: false,
		}
	}

	// Mark node as completed
	if nodeExec.Status == execution.NodeStatusRunning {
		nodeExec.Complete(nodeExec.Outputs)
	}

	// Add to execution record
	_ = exec.AddNodeExecution(nodeExec)

	// Log node execution
	if e.logger != nil {
		e.logger.LogNodeExecution(nodeExec)
	}

	return nil
}

// topologicalSort performs a topological sort on workflow nodes using Kahn's algorithm.
func (e *Engine) topologicalSort(wf *workflow.Workflow) ([]workflow.Node, error) {
	// Build adjacency list and in-degree count
	adjacency := make(map[string][]string)
	inDegree := make(map[string]int)

	// Initialize in-degree for all nodes
	for _, node := range wf.Nodes {
		nodeID := node.GetID()
		inDegree[nodeID] = 0
		adjacency[nodeID] = []string{}
	}

	// Build adjacency list and count in-degrees
	for _, edge := range wf.Edges {
		adjacency[edge.FromNodeID] = append(adjacency[edge.FromNodeID], edge.ToNodeID)
		inDegree[edge.ToNodeID]++
	}

	// Find all nodes with in-degree 0 (start nodes)
	queue := []string{}
	for nodeID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, nodeID)
		}
	}

	// Process nodes in topological order
	var sorted []workflow.Node
	nodeMap := make(map[string]workflow.Node)
	for _, node := range wf.Nodes {
		nodeMap[node.GetID()] = node
	}

	for len(queue) > 0 {
		// Dequeue
		currentID := queue[0]
		queue = queue[1:]

		// Add to sorted list
		sorted = append(sorted, nodeMap[currentID])

		// Reduce in-degree for neighbors
		for _, neighborID := range adjacency[currentID] {
			inDegree[neighborID]--
			if inDegree[neighborID] == 0 {
				queue = append(queue, neighborID)
			}
		}
	}

	// Check if all nodes were processed (no cycles)
	if len(sorted) != len(wf.Nodes) {
		return nil, fmt.Errorf("workflow contains a cycle")
	}

	return sorted, nil
}

// validateInputs checks that all required input variables are provided.
func (e *Engine) validateInputs(wf *workflow.Workflow, inputs map[string]interface{}) error {
	// For now, we don't have a Required field on Variable
	// This is a placeholder for future validation
	// All variables with no default are considered optional unless explicitly provided
	return nil
}

// initializeVariables sets up the execution context with workflow variables.
func (e *Engine) initializeVariables(ctx *execution.ExecutionContext, wf *workflow.Workflow) error {
	for _, variable := range wf.Variables {
		// Skip if already set by input
		if _, exists := ctx.GetVariable(variable.Name); exists {
			continue
		}

		// Set default value if provided
		if variable.DefaultValue != nil {
			if err := ctx.SetVariable(variable.Name, variable.DefaultValue); err != nil {
				return fmt.Errorf("failed to set default for variable '%s': %w", variable.Name, err)
			}
		}
	}
	return nil
}

// connectServers establishes connections to all MCP servers defined in the workflow.
func (e *Engine) connectServers(ctx context.Context, wf *workflow.Workflow) error {
	for _, serverConfig := range wf.ServerConfigs {
		// Create MCP server
		server, err := mcpserver.NewMCPServer(
			serverConfig.ID,
			serverConfig.Command,
			serverConfig.Args,
			mcpserver.TransportType(serverConfig.Transport),
		)
		if err != nil {
			return fmt.Errorf("failed to create server %s: %w", serverConfig.ID, err)
		}

		// Register server
		if err := e.serverRegistry.Register(server); err != nil {
			return fmt.Errorf("failed to register server %s: %w", serverConfig.ID, err)
		}

		// Connect to server
		if err := server.Connect(); err != nil {
			return fmt.Errorf("failed to connect to server %s: %w", serverConfig.ID, err)
		}

		// Complete connection
		if err := server.CompleteConnection(); err != nil {
			return fmt.Errorf("failed to complete connection to server %s: %w", serverConfig.ID, err)
		}

		// Discover available tools
		if err := server.DiscoverTools(); err != nil {
			return fmt.Errorf("failed to discover tools on server %s: %w", serverConfig.ID, err)
		}
	}

	return nil
}

// disconnectServers closes all server connections.
func (e *Engine) disconnectServers(wf *workflow.Workflow) {
	for _, serverConfig := range wf.ServerConfigs {
		if server, err := e.serverRegistry.Get(serverConfig.ID); err == nil {
			_ = server.Disconnect()
		}
	}
}

// Close cleans up engine resources.
func (e *Engine) Close() error {
	if e.execRepository != nil {
		return e.execRepository.Close()
	}
	return nil
}
