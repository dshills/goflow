package execution

import (
	"context"
	"fmt"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/transform"
	"github.com/dshills/goflow/pkg/workflow"
)

// LoopIteration represents the result of a single loop iteration
type LoopIteration struct {
	Index   int
	Item    interface{}
	Outputs map[string]interface{}
	Error   error
	Broken  bool // Whether loop was broken at this iteration
}

// executeLoopIterations executes the loop body for each item in the collection
func (e *Engine) executeLoopIterations(
	ctx context.Context,
	node *workflow.LoopNode,
	wf *workflow.Workflow,
	exec *execution.Execution,
	nodeMap map[string]workflow.Node,
) ([]LoopIteration, error) {
	// Get collection variable
	collection, exists := exec.Context.GetVariable(node.Collection)
	if !exists {
		return nil, fmt.Errorf("collection variable '%s' not found", node.Collection)
	}

	// Convert collection to slice
	items, err := convertToSlice(collection)
	if err != nil {
		return nil, fmt.Errorf("collection variable '%s' is not iterable: %w", node.Collection, err)
	}

	// Execute loop iterations
	iterations := make([]LoopIteration, 0, len(items))

	for index, item := range items {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return iterations, ctx.Err()
		default:
		}

		// Execute iteration
		iteration, broken, err := e.executeLoopIteration(
			ctx,
			node,
			wf,
			exec,
			nodeMap,
			index,
			item,
		)

		iterations = append(iterations, iteration)

		// Check if loop should break
		if err != nil {
			return iterations, err
		}

		if broken {
			break
		}
	}

	return iterations, nil
}

// executeLoopIteration executes a single iteration of the loop
func (e *Engine) executeLoopIteration(
	ctx context.Context,
	node *workflow.LoopNode,
	wf *workflow.Workflow,
	exec *execution.Execution,
	nodeMap map[string]workflow.Node,
	index int,
	item interface{},
) (LoopIteration, bool, error) {
	iteration := LoopIteration{
		Index:   index,
		Item:    item,
		Outputs: make(map[string]interface{}),
	}

	// Set item variable in execution context (scoped to this iteration)
	if err := exec.Context.SetVariable(node.ItemVariable, item); err != nil {
		iteration.Error = fmt.Errorf("failed to set item variable: %w", err)
		return iteration, false, iteration.Error
	}

	// Set loop index variable (optional convenience variable)
	indexVarName := node.ItemVariable + "_index"
	if err := exec.Context.SetVariable(indexVarName, index); err != nil {
		iteration.Error = fmt.Errorf("failed to set index variable: %w", err)
		return iteration, false, iteration.Error
	}

	// Check break condition BEFORE executing body (if specified)
	if node.BreakCondition != "" {
		broken, err := e.evaluateBreakCondition(node.BreakCondition, exec)
		if err != nil {
			iteration.Error = fmt.Errorf("break condition evaluation failed: %w", err)
			return iteration, false, iteration.Error
		}

		if broken {
			iteration.Broken = true
			return iteration, true, nil
		}
	}

	// Execute loop body nodes
	for _, nodeID := range node.Body {
		bodyNode, exists := nodeMap[nodeID]
		if !exists {
			err := fmt.Errorf("loop body node '%s' not found in workflow", nodeID)
			iteration.Error = err
			return iteration, false, err
		}

		// Execute the body node
		if err := e.executeNode(ctx, bodyNode, wf, exec); err != nil {
			iteration.Error = fmt.Errorf("loop body node '%s' failed: %w", nodeID, err)
			return iteration, false, iteration.Error
		}

		// Collect outputs
		if len(exec.NodeExecutions) > 0 {
			lastExec := exec.NodeExecutions[len(exec.NodeExecutions)-1]
			if lastExec.Outputs != nil {
				for k, v := range lastExec.Outputs {
					iteration.Outputs[k] = v
				}
			}
		}
	}

	return iteration, false, nil
}

// evaluateBreakCondition evaluates the break condition expression
// Returns true if the loop should break
func (e *Engine) evaluateBreakCondition(
	condition string,
	exec *execution.Execution,
) (bool, error) {
	// Create transformer for expression evaluation
	transformer := transform.NewTransformer()

	// Get all variables as context for expression evaluation
	contextData := exec.Context.CreateSnapshot()

	// Evaluate condition as boolean expression
	// Note: Using context.Background() since we don't have a cancellation context here
	result, err := transformer.Transform(context.Background(), condition, contextData)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate break condition: %w", err)
	}

	// Convert result to boolean
	broken, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("break condition must evaluate to boolean, got %T", result)
	}

	return broken, nil
}

// collectLoopResults aggregates results from all loop iterations
func (e *Engine) collectLoopResults(iterations []LoopIteration) map[string]interface{} {
	results := make(map[string]interface{})

	// Collect all outputs from iterations
	allOutputs := make([]map[string]interface{}, len(iterations))
	for i, iter := range iterations {
		allOutputs[i] = iter.Outputs
	}

	results["iterations"] = allOutputs
	results["iteration_count"] = len(iterations)

	// Check if any iteration was broken
	for _, iter := range iterations {
		if iter.Broken {
			results["broken"] = true
			results["break_index"] = iter.Index
			break
		}
	}

	return results
}

// convertToSlice converts various collection types to []interface{}
func convertToSlice(collection interface{}) ([]interface{}, error) {
	switch v := collection.(type) {
	case []interface{}:
		return v, nil
	case []string:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = item
		}
		return result, nil
	case []int:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = item
		}
		return result, nil
	case []float64:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = item
		}
		return result, nil
	case []bool:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = item
		}
		return result, nil
	case []map[string]interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = item
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported collection type: %T", collection)
	}
}
