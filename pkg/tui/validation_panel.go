package tui

import (
	"fmt"
	"sync"
	"time"
)

// ValidationError represents a blocking error in the workflow
type ValidationError struct {
	NodeID    string // Node with error ("" for global workflow errors)
	ErrorType string // Error category (e.g., "missing_field", "circular_dependency")
	Message   string // Human-readable error message
}

// ValidationWarning represents a non-blocking warning
type ValidationWarning struct {
	NodeID  string // Node with warning ("" for global warnings)
	Message string // Human-readable warning message
}

// ValidationStatus contains the results of workflow validation
// Thread-safe for async validation operations
type ValidationStatus struct {
	mu            sync.RWMutex        // Protects concurrent access
	IsValid       bool                // Overall validation status
	Errors        []ValidationError   // Blocking errors
	Warnings      []ValidationWarning // Non-blocking warnings
	LastValidated time.Time           // When validation last ran
}

// NewValidationStatus creates a new validation status
func NewValidationStatus() *ValidationStatus {
	return &ValidationStatus{
		IsValid:       true,
		Errors:        make([]ValidationError, 0),
		Warnings:      make([]ValidationWarning, 0),
		LastValidated: time.Time{}, // Zero time indicates never validated
	}
}

// AddError adds a validation error (thread-safe)
func (v *ValidationStatus) AddError(nodeID, errorType, message string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.Errors = append(v.Errors, ValidationError{
		NodeID:    nodeID,
		ErrorType: errorType,
		Message:   message,
	})
	v.IsValid = false
}

// AddWarning adds a validation warning (thread-safe)
func (v *ValidationStatus) AddWarning(nodeID, message string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.Warnings = append(v.Warnings, ValidationWarning{
		NodeID:  nodeID,
		Message: message,
	})
}

// Clear resets all validation results (thread-safe)
func (v *ValidationStatus) Clear() {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.IsValid = true
	v.Errors = make([]ValidationError, 0)
	v.Warnings = make([]ValidationWarning, 0)
	v.LastValidated = time.Time{}
}

// SetValidated marks validation as complete at current time (thread-safe)
func (v *ValidationStatus) SetValidated() {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.LastValidated = time.Now()
	// If no errors were added, mark as valid
	if len(v.Errors) == 0 {
		v.IsValid = true
	}
}

// GetErrors returns a copy of all errors (thread-safe)
func (v *ValidationStatus) GetErrors() []ValidationError {
	v.mu.RLock()
	defer v.mu.RUnlock()

	errors := make([]ValidationError, len(v.Errors))
	copy(errors, v.Errors)
	return errors
}

// GetWarnings returns a copy of all warnings (thread-safe)
func (v *ValidationStatus) GetWarnings() []ValidationWarning {
	v.mu.RLock()
	defer v.mu.RUnlock()

	warnings := make([]ValidationWarning, len(v.Warnings))
	copy(warnings, v.Warnings)
	return warnings
}

// ErrorCount returns the number of errors (thread-safe)
func (v *ValidationStatus) ErrorCount() int {
	v.mu.RLock()
	defer v.mu.RUnlock()

	return len(v.Errors)
}

// WarningCount returns the number of warnings (thread-safe)
func (v *ValidationStatus) WarningCount() int {
	v.mu.RLock()
	defer v.mu.RUnlock()

	return len(v.Warnings)
}

// HasErrors returns true if there are any errors (thread-safe)
func (v *ValidationStatus) HasErrors() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()

	return len(v.Errors) > 0
}

// HasWarnings returns true if there are any warnings (thread-safe)
func (v *ValidationStatus) HasWarnings() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()

	return len(v.Warnings) > 0
}

// GetNodeErrors returns all errors for a specific node (thread-safe)
func (v *ValidationStatus) GetNodeErrors(nodeID string) []ValidationError {
	v.mu.RLock()
	defer v.mu.RUnlock()

	nodeErrors := make([]ValidationError, 0)
	for _, err := range v.Errors {
		if err.NodeID == nodeID {
			nodeErrors = append(nodeErrors, err)
		}
	}
	return nodeErrors
}

// GetNodeWarnings returns all warnings for a specific node (thread-safe)
func (v *ValidationStatus) GetNodeWarnings(nodeID string) []ValidationWarning {
	v.mu.RLock()
	defer v.mu.RUnlock()

	nodeWarnings := make([]ValidationWarning, 0)
	for _, warn := range v.Warnings {
		if warn.NodeID == nodeID {
			nodeWarnings = append(nodeWarnings, warn)
		}
	}
	return nodeWarnings
}

// GetGlobalErrors returns all global errors (nodeID == "") (thread-safe)
func (v *ValidationStatus) GetGlobalErrors() []ValidationError {
	v.mu.RLock()
	defer v.mu.RUnlock()

	globalErrors := make([]ValidationError, 0)
	for _, err := range v.Errors {
		if err.NodeID == "" {
			globalErrors = append(globalErrors, err)
		}
	}
	return globalErrors
}

// GetGlobalWarnings returns all global warnings (nodeID == "") (thread-safe)
func (v *ValidationStatus) GetGlobalWarnings() []ValidationWarning {
	v.mu.RLock()
	defer v.mu.RUnlock()

	globalWarnings := make([]ValidationWarning, 0)
	for _, warn := range v.Warnings {
		if warn.NodeID == "" {
			globalWarnings = append(globalWarnings, warn)
		}
	}
	return globalWarnings
}

// Summary returns a human-readable validation summary (thread-safe)
func (v *ValidationStatus) Summary() string {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if v.IsValid {
		if len(v.Warnings) > 0 {
			return "Valid with warnings"
		}
		return "Valid"
	}

	errorCount := len(v.Errors)
	warningCount := len(v.Warnings)

	if warningCount > 0 {
		return fmt.Sprintf("%d error(s), %d warning(s)", errorCount, warningCount)
	}
	return fmt.Sprintf("%d error(s)", errorCount)
}

// ValidationPanel displays validation results in the TUI
type ValidationPanel struct {
	status        *ValidationStatus
	selectedIndex int  // Index of currently selected error/warning
	visible       bool // Whether panel is visible
}

// NewValidationPanel creates a validation panel with the given status
func NewValidationPanel(status *ValidationStatus) *ValidationPanel {
	return &ValidationPanel{
		status:        status,
		selectedIndex: 0,
		visible:       false,
	}
}

// Show opens the validation panel
func (p *ValidationPanel) Show() {
	p.visible = true
}

// Hide closes the validation panel
func (p *ValidationPanel) Hide() {
	p.visible = false
}

// IsVisible returns whether the panel is visible
func (p *ValidationPanel) IsVisible() bool {
	return p.visible
}

// Next selects the next error/warning in the list
func (p *ValidationPanel) Next() {
	if p.status == nil {
		return
	}

	totalItems := p.status.ErrorCount() + p.status.WarningCount()
	if totalItems == 0 {
		p.selectedIndex = 0
		return
	}

	p.selectedIndex = (p.selectedIndex + 1) % totalItems
}

// Previous selects the previous error/warning in the list
func (p *ValidationPanel) Previous() {
	if p.status == nil {
		return
	}

	totalItems := p.status.ErrorCount() + p.status.WarningCount()
	if totalItems == 0 {
		p.selectedIndex = 0
		return
	}

	p.selectedIndex--
	if p.selectedIndex < 0 {
		p.selectedIndex = totalItems - 1
	}
}

// GetSelectedNodeID returns the node ID of the currently selected error/warning
// Returns empty string if no selection or selection is a global error
func (p *ValidationPanel) GetSelectedNodeID() string {
	if p.status == nil {
		return ""
	}

	errors := p.status.GetErrors()
	warnings := p.status.GetWarnings()
	totalItems := len(errors) + len(warnings)

	if totalItems == 0 || p.selectedIndex >= totalItems {
		return ""
	}

	// Errors are listed first, then warnings
	if p.selectedIndex < len(errors) {
		return errors[p.selectedIndex].NodeID
	}

	warningIndex := p.selectedIndex - len(errors)
	if warningIndex < len(warnings) {
		return warnings[warningIndex].NodeID
	}

	return ""
}

// UpdateStatus updates the validation status and resets selection if needed
func (p *ValidationPanel) UpdateStatus(status *ValidationStatus) {
	p.status = status

	// Reset selection if it's out of bounds
	totalItems := 0
	if status != nil {
		totalItems = status.ErrorCount() + status.WarningCount()
	}

	if p.selectedIndex >= totalItems {
		p.selectedIndex = 0
	}
}

// GetStatus returns the current validation status
func (p *ValidationPanel) GetStatus() *ValidationStatus {
	return p.status
}
