package execution

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/workflow"
)

// MergeStrategy defines how parallel branches are synchronized
type MergeStrategy string

const (
	// MergeWaitAll waits for all branches to complete
	MergeWaitAll MergeStrategy = "wait_all"
	// MergeWaitAny waits for any branch to complete successfully
	MergeWaitAny MergeStrategy = "wait_any"
	// MergeWaitFirst waits for first branch to complete (success or failure)
	MergeWaitFirst MergeStrategy = "wait_first"
)

// BranchResult holds the result of executing a parallel branch
type BranchResult struct {
	BranchIndex int
	Outputs     map[string]interface{}
	Error       error
}

// executeParallelBranches executes multiple branches concurrently using the specified merge strategy
func (e *Engine) executeParallelBranches(
	ctx context.Context,
	node *workflow.ParallelNode,
	wf *workflow.Workflow,
	exec *execution.Execution,
	nodeMap map[string]workflow.Node,
) ([]BranchResult, error) {
	strategy := MergeStrategy(node.MergeStrategy)
	if strategy == "" {
		strategy = MergeWaitAll // Default strategy
	}

	numBranches := len(node.Branches)
	results := make([]BranchResult, numBranches)
	resultsMu := sync.Mutex{}

	// Create context for branch cancellation
	branchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	switch strategy {
	case MergeWaitAll:
		return e.executeWaitAll(branchCtx, cancel, node, wf, exec, nodeMap, results, &resultsMu)

	case MergeWaitAny:
		return e.executeWaitAny(branchCtx, cancel, node, wf, exec, nodeMap, results, &resultsMu)

	case MergeWaitFirst:
		return e.executeWaitFirst(branchCtx, cancel, node, wf, exec, nodeMap, results, &resultsMu)

	default:
		return nil, fmt.Errorf("unsupported merge strategy: %s", strategy)
	}
}

// executeWaitAll executes all branches and waits for all to complete
// Returns error if any branch fails
func (e *Engine) executeWaitAll(
	ctx context.Context,
	cancel context.CancelFunc,
	node *workflow.ParallelNode,
	wf *workflow.Workflow,
	exec *execution.Execution,
	nodeMap map[string]workflow.Node,
	results []BranchResult,
	resultsMu *sync.Mutex,
) ([]BranchResult, error) {
	g, gctx := errgroup.WithContext(ctx)

	// Store branch executions for later merging
	branchExecs := make([]*execution.Execution, len(node.Branches))

	for i, branch := range node.Branches {
		branchIndex := i
		branchNodes := branch

		g.Go(func() error {
			// Create isolated context for this branch
			branchExec, err := e.createBranchContext(exec)
			if err != nil {
				return fmt.Errorf("failed to create branch context: %w", err)
			}

			// Execute branch nodes
			branchOutputs, branchErr := e.executeBranchNodes(gctx, branchNodes, wf, branchExec, nodeMap)

			// Store result and branch execution
			resultsMu.Lock()
			results[branchIndex] = BranchResult{
				BranchIndex: branchIndex,
				Outputs:     branchOutputs,
				Error:       branchErr,
			}
			branchExecs[branchIndex] = branchExec
			resultsMu.Unlock()

			return branchErr
		})
	}

	// Wait for all branches to complete
	if err := g.Wait(); err != nil {
		return results, fmt.Errorf("parallel execution failed: %w", err)
	}

	// Merge all successful branch contexts back to parent (serially, after all branches complete)
	for i, branchExec := range branchExecs {
		if branchExec != nil && results[i].Error == nil {
			if err := e.mergeBranchContext(exec, branchExec); err != nil {
				return results, fmt.Errorf("failed to merge branch %d context: %w", i, err)
			}
		}
	}

	return results, nil
}

// executeWaitAny executes all branches but succeeds if any branch succeeds
// Returns error only if all branches fail
func (e *Engine) executeWaitAny(
	ctx context.Context,
	cancel context.CancelFunc,
	node *workflow.ParallelNode,
	wf *workflow.Workflow,
	exec *execution.Execution,
	nodeMap map[string]workflow.Node,
	results []BranchResult,
	resultsMu *sync.Mutex,
) ([]BranchResult, error) {
	var wg sync.WaitGroup
	successChan := make(chan int, 1) // Buffered to prevent goroutine leak
	allDone := make(chan struct{})
	branchExecs := make([]*execution.Execution, len(node.Branches))

	for i, branch := range node.Branches {
		branchIndex := i
		branchNodes := branch

		wg.Add(1)
		go func() {
			defer wg.Done()

			// Create isolated context for this branch
			branchExec, err := e.createBranchContext(exec)
			if err != nil {
				resultsMu.Lock()
				results[branchIndex] = BranchResult{
					BranchIndex: branchIndex,
					Error:       fmt.Errorf("failed to create branch context: %w", err),
				}
				resultsMu.Unlock()
				return
			}

			// Execute branch nodes
			branchOutputs, branchErr := e.executeBranchNodes(ctx, branchNodes, wf, branchExec, nodeMap)

			// Store result and branch execution
			resultsMu.Lock()
			results[branchIndex] = BranchResult{
				BranchIndex: branchIndex,
				Outputs:     branchOutputs,
				Error:       branchErr,
			}
			branchExecs[branchIndex] = branchExec
			resultsMu.Unlock()

			// If successful, signal success
			if branchErr == nil {
				select {
				case successChan <- branchIndex:
					// First success
				default:
					// Another branch already succeeded
				}
			}
		}()
	}

	// Wait for all branches in background
	go func() {
		wg.Wait()
		close(allDone)
	}()

	// Wait for first success or all failures
	select {
	case <-successChan:
		// At least one branch succeeded
		<-allDone // Wait for cleanup
	case <-allDone:
		// All branches completed, check if any succeeded
		hasSuccess := false
		for _, result := range results {
			if result.Error == nil {
				hasSuccess = true
				break
			}
		}
		if !hasSuccess {
			// All failed
			return results, fmt.Errorf("all parallel branches failed")
		}
	case <-ctx.Done():
		return results, ctx.Err()
	}

	// Merge all successful branch contexts back to parent (serially, after all branches complete)
	for i, branchExec := range branchExecs {
		if branchExec != nil && results[i].Error == nil {
			if err := e.mergeBranchContext(exec, branchExec); err != nil {
				// Log error but don't fail execution for wait_any strategy
				_ = err
			}
		}
	}

	return results, nil
}

// executeWaitFirst executes all branches but returns as soon as first completes
// Success or failure doesn't matter - just first to finish
func (e *Engine) executeWaitFirst(
	ctx context.Context,
	cancel context.CancelFunc,
	node *workflow.ParallelNode,
	wf *workflow.Workflow,
	exec *execution.Execution,
	nodeMap map[string]workflow.Node,
	results []BranchResult,
	resultsMu *sync.Mutex,
) ([]BranchResult, error) {
	var wg sync.WaitGroup
	firstDone := make(chan int, 1) // Buffered to prevent goroutine leak
	branchExecs := make([]*execution.Execution, len(node.Branches))

	for i, branch := range node.Branches {
		branchIndex := i
		branchNodes := branch

		wg.Add(1)
		go func() {
			defer wg.Done()

			// Create isolated context for this branch
			branchExec, err := e.createBranchContext(exec)
			if err != nil {
				resultsMu.Lock()
				results[branchIndex] = BranchResult{
					BranchIndex: branchIndex,
					Error:       fmt.Errorf("failed to create branch context: %w", err),
				}
				resultsMu.Unlock()

				// Signal completion even on error
				select {
				case firstDone <- branchIndex:
				default:
				}
				return
			}

			// Execute branch nodes
			branchOutputs, branchErr := e.executeBranchNodes(ctx, branchNodes, wf, branchExec, nodeMap)

			// Store result and branch execution
			resultsMu.Lock()
			results[branchIndex] = BranchResult{
				BranchIndex: branchIndex,
				Outputs:     branchOutputs,
				Error:       branchErr,
			}
			branchExecs[branchIndex] = branchExec
			resultsMu.Unlock()

			// Signal first completion
			select {
			case firstDone <- branchIndex:
			default:
				// Another branch already finished
			}
		}()
	}

	// Wait for first branch to complete
	var firstBranchIndex int
	select {
	case firstBranchIndex = <-firstDone:
		// Cancel other branches
		cancel()

		// Wait for all goroutines to clean up
		wg.Wait()
	case <-ctx.Done():
		cancel()
		wg.Wait()
		return results, ctx.Err()
	}

	// Merge the first completed branch context back to parent (if successful)
	if branchExecs[firstBranchIndex] != nil && results[firstBranchIndex].Error == nil {
		if err := e.mergeBranchContext(exec, branchExecs[firstBranchIndex]); err != nil {
			return results, fmt.Errorf("failed to merge first branch context: %w", err)
		}
	}

	// Return result from first branch
	return results, results[firstBranchIndex].Error
}

// createBranchContext creates an isolated execution context for a parallel branch
// The context inherits current variables but changes are isolated
func (e *Engine) createBranchContext(parentExec *execution.Execution) (*execution.Execution, error) {
	// Create a new execution with same workflow but isolated context
	branchExec, err := execution.NewExecution(
		parentExec.WorkflowID,
		parentExec.WorkflowVersion,
		nil, // No additional inputs
	)
	if err != nil {
		return nil, err
	}

	// Copy current variable state from parent
	parentExec.Context.CopyVariablesTo(branchExec.Context)

	return branchExec, nil
}

// mergeBranchContext merges variable changes and node executions from branch back to parent context
func (e *Engine) mergeBranchContext(parentExec *execution.Execution, branchExec *execution.Execution) error {
	// Copy all variables from branch context to parent context
	branchExec.Context.CopyVariablesTo(parentExec.Context)

	// Merge node executions from branch to parent
	// This allows the parent execution to track all nodes executed in branches
	parentExec.NodeExecutions = append(parentExec.NodeExecutions, branchExec.NodeExecutions...)

	// Note: Variable history from branch is not merged to avoid cluttering audit trail
	// Each branch maintains its own history

	return nil
}

// executeBranchNodes executes a sequence of nodes in a parallel branch
func (e *Engine) executeBranchNodes(
	ctx context.Context,
	nodeIDs []string,
	wf *workflow.Workflow,
	branchExec *execution.Execution,
	nodeMap map[string]workflow.Node,
) (map[string]interface{}, error) {
	outputs := make(map[string]interface{})

	for _, nodeID := range nodeIDs {
		node, exists := nodeMap[nodeID]
		if !exists {
			return outputs, fmt.Errorf("node '%s' not found in workflow", nodeID)
		}

		// Check for context cancellation
		select {
		case <-ctx.Done():
			return outputs, ctx.Err()
		default:
		}

		// Execute the node
		if err := e.executeNode(ctx, node, wf, branchExec); err != nil {
			return outputs, fmt.Errorf("node '%s' failed: %w", nodeID, err)
		}

		// Collect outputs
		if len(branchExec.NodeExecutions) > 0 {
			lastExec := branchExec.NodeExecutions[len(branchExec.NodeExecutions)-1]
			if lastExec.Outputs != nil {
				for k, v := range lastExec.Outputs {
					outputs[k] = v
				}
			}
		}
	}

	return outputs, nil
}
