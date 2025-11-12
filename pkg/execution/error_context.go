package execution

import (
	"github.com/dshills/goflow/pkg/errors"
)

// OperationalError is an alias for errors.OperationalError for backward compatibility.
type OperationalError = errors.OperationalError

// NewOperationalError creates an OperationalError wrapping an error.
//
// Re-exported from pkg/errors for convenience.
//
// Returns nil if cause is nil (no error to wrap).
//
// Example:
//
//	if err != nil {
//	    return NewOperationalError("executing node", workflowID, nodeID, err)
//	}
func NewOperationalError(operation, workflowID, nodeID string, cause error) *OperationalError {
	return errors.NewOperationalError(operation, workflowID, nodeID, cause)
}

// NewOperationalErrorWithAttrs creates an OperationalError with additional attributes.
//
// Re-exported from pkg/errors for convenience.
//
// Returns nil if cause is nil (no error to wrap).
//
// Example:
//
//	return NewOperationalErrorWithAttrs(
//	    "validating input",
//	    workflowID,
//	    nodeID,
//	    err,
//	    map[string]interface{}{
//	        "inputSize": len(input),
//	        "schema":    schemaName,
//	    },
//	)
func NewOperationalErrorWithAttrs(operation, workflowID, nodeID string, cause error, attrs map[string]interface{}) *OperationalError {
	return errors.NewOperationalErrorWithAttrs(operation, workflowID, nodeID, cause, attrs)
}
