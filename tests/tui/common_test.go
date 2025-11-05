package tui

import (
	tui "github.com/dshills/goflow/pkg/tui"
)

// Common test types and helpers
// Type aliases for types defined in the tui package
type Position = tui.Position

// Edge represents a workflow edge (for test purposes)
type Edge struct {
	From string
	To   string
}
