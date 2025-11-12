package execution

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/transform"
	"github.com/dshills/goflow/pkg/workflow"
)

// executeStartNode executes a Start node (entry point of workflow).
func (e *Engine) executeStartNode(ctx context.Context, node *workflow.StartNode, exec *execution.Execution, nodeExec *execution.NodeExecution) error {
	// Start node just marks the beginning of execution
	// No actual work to do here
	nodeExec.Outputs = map[string]interface{}{
		"started_at": nodeExec.StartedAt,
	}

	return nil
}

// executeEndNode executes an End node (exit point of workflow).
func (e *Engine) executeEndNode(ctx context.Context, node *workflow.EndNode, exec *execution.Execution, nodeExec *execution.NodeExecution) error {
	// Evaluate return expression if present
	var returnValue interface{}
	if node.ReturnValue != "" {
		// Substitute variables in return expression
		expr, err := e.substituteVariables(node.ReturnValue, exec.Context)
		if err != nil {
			return fmt.Errorf("failed to substitute variables in return expression: %w", err)
		}

		// If the expression is a simple variable reference, get its value
		if strings.HasPrefix(expr, "${") && strings.HasSuffix(expr, "}") {
			varName := strings.TrimSuffix(strings.TrimPrefix(expr, "${"), "}")
			value, exists := exec.Context.GetVariable(varName)
			if !exists {
				return fmt.Errorf("return variable '%s' not found", varName)
			}
			returnValue = value
		} else {
			// Otherwise, use the substituted string as return value
			returnValue = expr
		}
	}

	// Set return value in execution
	exec.ReturnValue = returnValue

	// Set outputs for logging
	nodeExec.Outputs = map[string]interface{}{
		"return_value": returnValue,
		"completed_at": nodeExec.CompletedAt,
	}

	return nil
}

// executeMCPToolNode executes an MCP tool node.
func (e *Engine) executeMCPToolNode(ctx context.Context, node *workflow.MCPToolNode, wf *workflow.Workflow, exec *execution.Execution, nodeExec *execution.NodeExecution) error {
	// Get MCP server
	server, err := e.serverRegistry.Get(node.ServerID)
	if err != nil {
		return fmt.Errorf("server '%s' not found: %w", node.ServerID, err)
	}

	// Substitute variables in parameters
	params := make(map[string]interface{})
	for key, value := range node.Parameters {
		substituted, err := e.substituteVariables(value, exec.Context)
		if err != nil {
			return fmt.Errorf("failed to substitute variables in parameter '%s': %w", key, err)
		}
		params[key] = substituted
	}

	// Record inputs
	nodeExec.Inputs = params

	// Invoke tool
	result, err := server.InvokeTool(node.ToolName, params)
	if err != nil {
		// Check if it's a recoverable error
		recoverable := strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "connection")

		return &MCPToolError{
			ServerID:    node.ServerID,
			ToolName:    node.ToolName,
			Message:     fmt.Sprintf("tool invocation failed: %v", err),
			Recoverable: recoverable,
			Context: map[string]interface{}{
				"parameters": params,
			},
		}
	}

	// Store result in context
	if node.OutputVariable != "" {
		if err := exec.Context.SetVariableWithNode(node.OutputVariable, result, nodeExec.ID); err != nil {
			return fmt.Errorf("failed to set output variable '%s': %w", node.OutputVariable, err)
		}

		// Log variable change
		if e.logger != nil {
			snapshots := exec.Context.GetVariableHistory()
			if len(snapshots) > 0 {
				e.logger.LogVariableChange(&snapshots[len(snapshots)-1])
			}
		}
	}

	// Record outputs
	nodeExec.Outputs = map[string]interface{}{
		node.OutputVariable: result,
	}

	return nil
}

// executeTransformNode executes a Transform node.
func (e *Engine) executeTransformNode(ctx context.Context, node *workflow.TransformNode, exec *execution.Execution, nodeExec *execution.NodeExecution) error {
	// Get input variable value
	inputValue, exists := exec.Context.GetVariable(node.InputVariable)
	if !exists {
		return fmt.Errorf("input variable '%s' not found", node.InputVariable)
	}

	// Record inputs
	nodeExec.Inputs = map[string]interface{}{
		node.InputVariable: inputValue,
	}

	// Create transformer
	transformer := transform.NewTransformer()

	// Determine what to pass to the transformer based on expression type
	// JSONPath queries operate on the input value directly
	// Expression/Template evaluations need the full variable context
	var transformData interface{}
	if e.isJSONPathExpression(node.Expression) {
		// For JSONPath: pass the input value (data to query)
		transformData = inputValue
	} else {
		// For Expression/Template: pass the full context snapshot
		// This allows expressions to reference any variable
		transformData = exec.Context.CreateSnapshot()
	}

	// Apply transformation
	result, err := transformer.Transform(ctx, node.Expression, transformData)
	if err != nil {
		return &TransformError{
			InputVariable: node.InputVariable,
			Expression:    node.Expression,
			Message:       fmt.Sprintf("transformation failed: %v", err),
			Context: map[string]interface{}{
				"input_value": inputValue,
				"expression":  node.Expression,
			},
		}
	}

	// Store result in context
	if err := exec.Context.SetVariableWithNode(node.OutputVariable, result, nodeExec.ID); err != nil {
		return fmt.Errorf("failed to set output variable '%s': %w", node.OutputVariable, err)
	}

	// Log variable change
	if e.logger != nil {
		snapshots := exec.Context.GetVariableHistory()
		if len(snapshots) > 0 {
			e.logger.LogVariableChange(&snapshots[len(snapshots)-1])
		}
	}

	// Record outputs
	nodeExec.Outputs = map[string]interface{}{
		node.OutputVariable: result,
	}

	return nil
}

// isJSONPathExpression determines if an expression is a JSONPath query
// This duplicates the detection logic from transform.detectTransformType for JSONPath
func (e *Engine) isJSONPathExpression(expr string) bool {
	trimmed := strings.TrimSpace(expr)

	// Check for JSONPath patterns
	if strings.HasPrefix(trimmed, "$.") || trimmed == "$" {
		return true
	}

	// Check for recursive descent
	if strings.Contains(trimmed, "..") {
		return true
	}

	// Check for filter expressions
	if strings.Contains(trimmed, "[?(") {
		return true
	}

	// Check for array wildcard
	if strings.Contains(trimmed, "[*]") {
		return true
	}

	return false
}

// substituteVariables replaces variable placeholders (${var_name}) with actual values from context.
// resolveVariablePath resolves a variable path like "user.name" or "config.database.host"
// Supports nested field access via dot notation for map[string]interface{} values only
// Note: Does not currently support array/slice indexing with brackets (e.g., items[0])
func (e *Engine) resolveVariablePath(ctx *execution.ExecutionContext, path string) (interface{}, error) {
	// Split path by dots
	parts := strings.Split(path, ".")

	// Get the root variable
	value, exists := ctx.GetVariable(parts[0])
	if !exists {
		return nil, fmt.Errorf("variable '%s' not found", parts[0])
	}

	// Navigate through nested fields
	current := value
	for i := 1; i < len(parts); i++ {
		field := parts[i]

		// Handle map access
		if m, ok := current.(map[string]interface{}); ok {
			val, exists := m[field]
			if !exists {
				return nil, fmt.Errorf("field '%s' not found in variable '%s'", field, strings.Join(parts[:i+1], "."))
			}
			current = val
			continue
		}

		// If not a map, can't access nested fields
		return nil, fmt.Errorf("cannot access field '%s' on non-map value (type: %T)", field, current)
	}

	return current, nil
}

func (e *Engine) substituteVariables(input string, ctx *execution.ExecutionContext) (string, error) {
	// Pattern to match ${variable_name}
	pattern := regexp.MustCompile(`\$\{([^}]+)\}`)

	// Find all matches
	matches := pattern.FindAllStringSubmatch(input, -1)

	result := input
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		placeholder := match[0] // Full match: ${var_name}
		varPath := match[1]     // Variable path: var_name or var_name.field.subfield

		// Get variable value (supports nested field access via dot notation)
		value, err := e.resolveVariablePath(ctx, varPath)
		if err != nil {
			return "", err
		}

		// Convert value to string
		var strValue string
		switch v := value.(type) {
		case string:
			strValue = v
		case nil:
			strValue = ""
		default:
			strValue = fmt.Sprintf("%v", v)
		}

		// Replace placeholder with value
		result = strings.Replace(result, placeholder, strValue, -1)
	}

	return result, nil
}

// MCPToolError represents an error during MCP tool execution.
type MCPToolError struct {
	ServerID    string
	ToolName    string
	Message     string
	Recoverable bool
	Context     map[string]interface{}
}

// Error implements the error interface.
func (e *MCPToolError) Error() string {
	return fmt.Sprintf("MCP tool error [%s/%s]: %s", e.ServerID, e.ToolName, e.Message)
}

// executeConditionNode executes a Condition node by evaluating its expression.
func (e *Engine) executeConditionNode(ctx context.Context, node *workflow.ConditionNode, exec *execution.Execution, nodeExec *execution.NodeExecution) error {
	// Prepare evaluation context with current variables
	evalContext := exec.Context.CreateSnapshot()

	// Process the condition expression to handle JSONPath-like syntax
	processedExpr, err := e.processConditionExpression(ctx, node.Condition, evalContext)
	if err != nil {
		return &ConditionError{
			Expression: node.Condition,
			Message:    fmt.Sprintf("failed to process condition expression: %v", err),
			Context: map[string]interface{}{
				"variables": evalContext,
			},
		}
	}

	// Create expression evaluator
	evaluator := transform.NewExpressionEvaluator()

	// Evaluate the condition expression
	result, err := evaluator.Evaluate(ctx, processedExpr, evalContext)
	if err != nil {
		return &ConditionError{
			Expression: node.Condition,
			Message:    fmt.Sprintf("condition evaluation failed: %v", err),
			Context: map[string]interface{}{
				"variables": evalContext,
			},
		}
	}

	// Ensure result is boolean
	boolResult, ok := result.(bool)
	if !ok {
		return &ConditionError{
			Expression: node.Condition,
			Message:    fmt.Sprintf("condition expression did not evaluate to boolean, got %T", result),
			Context: map[string]interface{}{
				"result": result,
			},
		}
	}

	// Record the condition result in outputs
	nodeExec.Outputs = map[string]interface{}{
		"result":    boolResult,
		"condition": node.Condition,
	}

	return nil
}

// processConditionExpression converts JSONPath-style expressions ($.variable) into
// direct variable references for evaluation. For example:
// "$.fileSize > 1048576" becomes "fileSize > 1048576"
func (e *Engine) processConditionExpression(ctx context.Context, expression string, variables map[string]interface{}) (string, error) {
	// Use regex to find all $.variable patterns
	pattern := regexp.MustCompile(`\$\.([a-zA-Z_][a-zA-Z0-9_]*)`)

	result := pattern.ReplaceAllStringFunc(expression, func(match string) string {
		// Extract variable name (remove "$." prefix)
		varName := match[2:]

		// Check if variable exists
		if _, exists := variables[varName]; exists {
			// Return the variable name without the "$." prefix
			return varName
		}
		// If variable doesn't exist, return as-is and let evaluator handle the error
		return match
	})

	return result, nil
}

// executeParallelNode executes a Parallel node with concurrent branch execution.
func (e *Engine) executeParallelNode(ctx context.Context, node *workflow.ParallelNode, wf *workflow.Workflow, exec *execution.Execution, nodeExec *execution.NodeExecution) error {
	// Create node map for quick lookup
	nodeMap := make(map[string]workflow.Node)
	for _, n := range wf.Nodes {
		nodeMap[n.GetID()] = n
	}

	// Record inputs
	nodeExec.Inputs = map[string]interface{}{
		"branches":       node.Branches,
		"merge_strategy": node.MergeStrategy,
	}

	// Execute parallel branches
	results, err := e.executeParallelBranches(ctx, node, wf, exec, nodeMap)
	if err != nil {
		return &ParallelExecutionError{
			NodeID:        node.ID,
			MergeStrategy: node.MergeStrategy,
			Message:       fmt.Sprintf("parallel execution failed: %v", err),
			BranchErrors:  collectBranchErrors(results),
		}
	}

	// Collect outputs from all branches
	branchOutputs := make([]map[string]interface{}, len(results))
	for i, result := range results {
		branchOutputs[i] = result.Outputs
	}

	nodeExec.Outputs = map[string]interface{}{
		"branches":     branchOutputs,
		"branch_count": len(results),
	}

	return nil
}

// executeLoopNode executes a Loop node with iteration over a collection.
func (e *Engine) executeLoopNode(ctx context.Context, node *workflow.LoopNode, wf *workflow.Workflow, exec *execution.Execution, nodeExec *execution.NodeExecution) error {
	// Create node map for quick lookup
	nodeMap := make(map[string]workflow.Node)
	for _, n := range wf.Nodes {
		nodeMap[n.GetID()] = n
	}

	// Get collection value for recording
	collection, exists := exec.Context.GetVariable(node.Collection)
	if !exists {
		return fmt.Errorf("collection variable '%s' not found", node.Collection)
	}

	// Record inputs
	nodeExec.Inputs = map[string]interface{}{
		"collection":      collection,
		"item_variable":   node.ItemVariable,
		"body":            node.Body,
		"break_condition": node.BreakCondition,
	}

	// Execute loop iterations
	iterations, err := e.executeLoopIterations(ctx, node, wf, exec, nodeMap)
	if err != nil {
		return &LoopExecutionError{
			NodeID:         node.ID,
			Collection:     node.Collection,
			Message:        fmt.Sprintf("loop execution failed: %v", err),
			IterationIndex: len(iterations) - 1,
			IterationError: err,
		}
	}

	// Clean up loop variables to prevent leakage outside loop scope
	exec.Context.DeleteVariable(node.ItemVariable)
	indexVarName := node.ItemVariable + "_index"
	exec.Context.DeleteVariable(indexVarName)

	// Collect and record results
	loopResults := e.collectLoopResults(iterations)
	nodeExec.Outputs = loopResults

	return nil
}

// collectBranchErrors extracts errors from branch results for error reporting
func collectBranchErrors(results []BranchResult) map[int]error {
	errors := make(map[int]error)
	for _, result := range results {
		if result.Error != nil {
			errors[result.BranchIndex] = result.Error
		}
	}
	return errors
}

// TransformError represents an error during data transformation.
type TransformError struct {
	InputVariable string
	Expression    string
	Message       string
	Context       map[string]interface{}
}

// Error implements the error interface.
func (e *TransformError) Error() string {
	return fmt.Sprintf("transform error [%s]: %s", e.InputVariable, e.Message)
}

// ConditionError represents an error during condition evaluation.
type ConditionError struct {
	Expression string
	Message    string
	Context    map[string]interface{}
}

// Error implements the error interface.
func (e *ConditionError) Error() string {
	return fmt.Sprintf("condition error [%s]: %s", e.Expression, e.Message)
}

// ParallelExecutionError represents an error during parallel execution.
type ParallelExecutionError struct {
	NodeID        string
	MergeStrategy string
	Message       string
	BranchErrors  map[int]error
}

// Error implements the error interface.
func (e *ParallelExecutionError) Error() string {
	return fmt.Sprintf("parallel execution error [node=%s, strategy=%s]: %s", e.NodeID, e.MergeStrategy, e.Message)
}

// LoopExecutionError represents an error during loop execution.
type LoopExecutionError struct {
	NodeID         string
	Collection     string
	Message        string
	IterationIndex int
	IterationError error
}

// Error implements the error interface.
func (e *LoopExecutionError) Error() string {
	return fmt.Sprintf("loop execution error [node=%s, collection=%s, iteration=%d]: %s",
		e.NodeID, e.Collection, e.IterationIndex, e.Message)
}
