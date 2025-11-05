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

	// Apply transformation
	result, err := transformer.Transform(ctx, node.Expression, inputValue)
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

// substituteVariables replaces variable placeholders (${var_name}) with actual values from context.
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
		varName := match[1]     // Variable name: var_name

		// Get variable value
		value, exists := ctx.GetVariable(varName)
		if !exists {
			return "", fmt.Errorf("variable '%s' not found", varName)
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
