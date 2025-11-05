package execution

import (
	"time"

	"github.com/dshills/goflow/pkg/domain/types"
)

// VariableSnapshot represents a point-in-time capture of a variable value change.
// It is used to maintain an append-only audit trail of all variable modifications.
// Once created, snapshots are immutable.
type VariableSnapshot struct {
	// Timestamp records when the variable value changed.
	Timestamp time.Time
	// NodeExecutionID identifies which node execution made the change (empty string if not from a node).
	NodeExecutionID types.NodeExecutionID
	// VariableName is the name of the variable that changed.
	VariableName string
	// OldValue is the previous value (nil if this is the first assignment).
	OldValue interface{}
	// NewValue is the new value after the change.
	NewValue interface{}
}

// NewVariableSnapshot creates a new variable snapshot.
func NewVariableSnapshot(
	variableName string,
	oldValue, newValue interface{},
	nodeExecID types.NodeExecutionID,
) VariableSnapshot {
	return VariableSnapshot{
		Timestamp:       time.Now(),
		NodeExecutionID: nodeExecID,
		VariableName:    variableName,
		OldValue:        oldValue,
		NewValue:        newValue,
	}
}
