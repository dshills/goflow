package cli

import (
	"fmt"
	"strings"

	"github.com/dshills/goflow/pkg/mcpserver"
	"github.com/dshills/goflow/pkg/workflow"
)

// MissingServerError indicates that workflow references servers not in the registry
type MissingServerError struct {
	MissingServers []string
}

func (e *MissingServerError) Error() string {
	return fmt.Sprintf("workflow references %d missing server(s): %s",
		len(e.MissingServers), strings.Join(e.MissingServers, ", "))
}

// CredentialPlaceholderWarning indicates servers with credential placeholders
type CredentialPlaceholderWarning struct {
	ServersWithPlaceholders []string
}

func (w *CredentialPlaceholderWarning) Error() string {
	return fmt.Sprintf("workflow contains %d server(s) with credential placeholders: %s",
		len(w.ServersWithPlaceholders), strings.Join(w.ServersWithPlaceholders, ", "))
}

// IncompatibleVersionError indicates workflow version is not supported
type IncompatibleVersionError struct {
	WorkflowVersion   string
	SupportedVersions []string
}

func (e *IncompatibleVersionError) Error() string {
	return fmt.Sprintf("workflow version %s is not compatible (supported versions: %s)",
		e.WorkflowVersion, strings.Join(e.SupportedVersions, ", "))
}

// ImportWorkflow imports a workflow from a file and validates server references
// against the provided MCP server registry.
//
// The import process:
// 1. Load and parse workflow YAML file
// 2. Validate workflow version compatibility
// 3. Check all server references against registry
// 4. Detect credential placeholders
// 5. Validate workflow structure
// 6. Return workflow with appropriate errors/warnings
//
// Returns:
// - (*Workflow, nil) if import is fully successful
// - (*Workflow, error) if there are validation errors (workflow still returned for inspection)
// - (nil, error) if the file cannot be read or parsed
func ImportWorkflow(path string, registry mcpserver.ServerRepository) (*workflow.Workflow, error) {
	// Load workflow from file
	wf, err := LoadWorkflowFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load workflow: %w", err)
	}

	// Check for empty workflow
	if wf.Name == "" && wf.Version == "" {
		return nil, fmt.Errorf("empty workflow file")
	}

	// Validate workflow version
	if err := validateWorkflowVersion(wf.Version); err != nil {
		return wf, err
	}

	// Validate server references against registry
	if err := validateServerReferences(wf, registry); err != nil {
		return wf, err
	}

	// Check for credential placeholders
	if err := checkCredentialPlaceholders(wf); err != nil {
		// This is a warning, not a hard error
		// Return workflow with warning
		return wf, err
	}

	// Validate workflow structure
	if err := wf.Validate(); err != nil {
		return wf, fmt.Errorf("workflow validation failed: %w", err)
	}

	return wf, nil
}

// validateWorkflowVersion checks if the workflow version is supported
func validateWorkflowVersion(version string) error {
	supportedVersions := []string{"1.0", "1.0.0"}

	// Normalize version for comparison
	normalized := strings.TrimPrefix(version, "v")

	for _, supported := range supportedVersions {
		if normalized == supported {
			return nil
		}
	}

	// Check if it's a compatible minor version (1.x)
	if strings.HasPrefix(normalized, "1.") {
		return nil
	}

	return &IncompatibleVersionError{
		WorkflowVersion:   version,
		SupportedVersions: supportedVersions,
	}
}

// validateServerReferences checks that all servers referenced in the workflow
// are registered in the MCP server registry
func validateServerReferences(wf *workflow.Workflow, registry mcpserver.ServerRepository) error {
	// Collect all server IDs referenced in workflow
	referencedServers := make(map[string]bool)

	// Get servers from server configs
	for _, sc := range wf.ServerConfigs {
		referencedServers[sc.ID] = true
	}

	// Also check servers referenced in MCP tool nodes
	for _, node := range wf.Nodes {
		if mcpNode, ok := node.(*workflow.MCPToolNode); ok {
			if mcpNode.ServerID != "" {
				referencedServers[mcpNode.ServerID] = true
			}
		}
	}

	// Check each referenced server against registry
	var missingServers []string
	for serverID := range referencedServers {
		_, err := registry.Get(serverID)
		if err != nil {
			// Server not found in registry
			missingServers = append(missingServers, serverID)
		}
	}

	if len(missingServers) > 0 {
		return &MissingServerError{
			MissingServers: missingServers,
		}
	}

	return nil
}

// checkCredentialPlaceholders detects servers with credential placeholders
// that need to be resolved before execution
func checkCredentialPlaceholders(wf *workflow.Workflow) error {
	var serversWithPlaceholders []string

	for _, sc := range wf.ServerConfigs {
		if hasCredentialPlaceholder(sc.CredentialRef) {
			serversWithPlaceholders = append(serversWithPlaceholders, sc.ID)
		}
	}

	if len(serversWithPlaceholders) > 0 {
		return &CredentialPlaceholderWarning{
			ServersWithPlaceholders: serversWithPlaceholders,
		}
	}

	return nil
}

// hasCredentialPlaceholder checks if a credential reference contains a placeholder
func hasCredentialPlaceholder(credentialRef string) bool {
	if credentialRef == "" {
		return false
	}

	// Check for common placeholder patterns
	// {{VAR_NAME}} - double curly braces
	// ${VAR_NAME} - shell-style
	// $VAR_NAME - simple variable
	// <PLACEHOLDER> - angle brackets

	placeholderPatterns := []string{
		"{{", // double curly braces start
		"${", // shell-style start
		"<",  // angle bracket start
	}

	for _, pattern := range placeholderPatterns {
		if strings.Contains(credentialRef, pattern) {
			return true
		}
	}

	// Check if it starts with $ (simple variable)
	if strings.HasPrefix(credentialRef, "$") && !strings.HasPrefix(credentialRef, "${") {
		return true
	}

	return false
}
