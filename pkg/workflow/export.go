package workflow

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// sensitiveEnvKeyPatterns are patterns that indicate sensitive credentials
var sensitiveEnvKeyPatterns = []string{
	"KEY",
	"SECRET",
	"TOKEN",
	"PASSWORD",
	"PASSPHRASE",
	"CREDENTIAL",
	"AUTH",
	"BEARER",
	"PRIVATE",
	"CLIENT_SECRET",
}

// Export exports a workflow to YAML format with credentials stripped
// for sharing. It performs the following:
//   - Deep copies the workflow to avoid mutating the original
//   - Removes sensitive environment variables from server configs
//   - Replaces credential_ref with placeholder
//   - Converts to YAML using ToYAML()
//
// Sensitive environment variables are detected by checking if the key name
// contains any of the sensitive patterns (KEY, SECRET, TOKEN, PASSWORD, etc.)
//
// Non-sensitive environment variables (HOST, PORT, LOG_LEVEL, SERVICE_NAME, etc.)
// are preserved.
//
// SECURITY LIMITATIONS:
//   - Currently only sanitizes server config environment variables
//   - Does not inspect environment variable values for embedded credentials
//   - Does not scan other fields (node configs, metadata, etc.) for secrets
//   - Users should review exported YAML before sharing externally
//
// Returns the workflow as YAML bytes suitable for sharing, or an error if
// the workflow is invalid or cannot be serialized.
func Export(workflow *Workflow) ([]byte, error) {
	if workflow == nil {
		return nil, errors.New("workflow cannot be nil")
	}

	// Deep copy the workflow to avoid mutating the original
	workflowCopy := deepCopyWorkflow(workflow)

	// Strip credentials from server configs
	for _, sc := range workflowCopy.ServerConfigs {
		// Strip sensitive environment variables
		if len(sc.Env) > 0 {
			sanitizedEnv := make(map[string]string)
			for key, value := range sc.Env {
				if !isSensitiveEnvKey(key) {
					// Keep non-sensitive env vars
					sanitizedEnv[key] = value
				}
			}

			sc.Env = sanitizedEnv
		}

		// Replace credential_ref with placeholder if present
		if sc.CredentialRef != "" {
			sc.CredentialRef = "<CREDENTIAL_REF_REQUIRED>"
		}
	}

	// Convert to YAML
	yamlBytes, err := ToYAML(workflowCopy)
	if err != nil {
		return nil, fmt.Errorf("failed to convert workflow to YAML: %w", err)
	}

	// Add credential warning comment at the top
	comment := "# CREDENTIAL WARNING: This workflow has been exported for sharing.\n" +
		"# Sensitive credentials have been removed and must be configured before use.\n" +
		"# Please set up credential references for servers marked with <CREDENTIAL_REF_REQUIRED>.\n\n"

	return append([]byte(comment), yamlBytes...), nil
}

// ExportFile exports a workflow to a YAML file with credentials stripped
func ExportFile(workflow *Workflow, path string) error {
	if workflow == nil {
		return errors.New("workflow cannot be nil")
	}
	if path == "" {
		return errors.New("file path cannot be empty")
	}

	// Export to YAML bytes
	yamlBytes, err := Export(workflow)
	if err != nil {
		return fmt.Errorf("failed to export workflow: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, yamlBytes, 0644); err != nil {
		return fmt.Errorf("failed to write workflow file: %w", err)
	}

	return nil
}

// isSensitiveEnvKey checks if an environment variable key contains sensitive information
func isSensitiveEnvKey(key string) bool {
	upperKey := strings.ToUpper(key)

	// Check for sensitive patterns
	for _, pattern := range sensitiveEnvKeyPatterns {
		if strings.Contains(upperKey, pattern) {
			return true
		}
	}

	// Special case: DATABASE_URL often contains credentials
	if strings.Contains(upperKey, "DATABASE") && strings.Contains(upperKey, "URL") {
		return true
	}
	if strings.Contains(upperKey, "CONN") && strings.Contains(upperKey, "STRING") {
		return true
	}
	if strings.Contains(upperKey, "DSN") { // Data Source Name
		return true
	}

	// Check for OAuth patterns
	if strings.Contains(upperKey, "OAUTH") {
		return true
	}

	return false
}

// deepCopyWorkflow creates a deep copy of a workflow to avoid mutating the original
func deepCopyWorkflow(wf *Workflow) *Workflow {
	if wf == nil {
		return nil
	}

	// Create new workflow
	wfCopy := &Workflow{
		ID:          wf.ID,
		Name:        wf.Name,
		Version:     wf.Version,
		Description: wf.Description,
		Metadata: WorkflowMetadata{
			Author:       wf.Metadata.Author,
			Created:      wf.Metadata.Created,
			LastModified: wf.Metadata.LastModified,
			Tags:         append([]string(nil), wf.Metadata.Tags...),
			Icon:         wf.Metadata.Icon,
		},
	}

	// Deep copy variables
	// NOTE: Variable.DefaultValue is interface{} and may contain complex types.
	// Current implementation performs shallow copy. For complete safety, DefaultValue
	// should be deep copied based on its concrete type.
	wfCopy.Variables = make([]*Variable, len(wf.Variables))
	for i, v := range wf.Variables {
		if v != nil {
			wfCopy.Variables[i] = &Variable{
				Name:         v.Name,
				Type:         v.Type,
				DefaultValue: v.DefaultValue, // Shallow copy - see note above
				Description:  v.Description,
			}
		}
	}

	// Deep copy server configs
	wfCopy.ServerConfigs = make([]*ServerConfig, len(wf.ServerConfigs))
	for i, sc := range wf.ServerConfigs {
		if sc != nil {
			wfCopy.ServerConfigs[i] = &ServerConfig{
				ID:            sc.ID,
				Name:          sc.Name,
				Command:       sc.Command,
				Args:          append([]string(nil), sc.Args...),
				Transport:     sc.Transport,
				Env:           deepCopyStringMap(sc.Env),
				CredentialRef: sc.CredentialRef,
			}
		}
	}

	// Deep copy nodes (shallow copy is sufficient as nodes are value types)
	wfCopy.Nodes = append([]Node(nil), wf.Nodes...)

	// Deep copy edges
	wfCopy.Edges = make([]*Edge, len(wf.Edges))
	for i, e := range wf.Edges {
		if e != nil {
			wfCopy.Edges[i] = &Edge{
				ID:         e.ID,
				FromNodeID: e.FromNodeID,
				ToNodeID:   e.ToNodeID,
				Condition:  e.Condition,
				Label:      e.Label,
			}
		}
	}

	return wfCopy
}

// deepCopyStringMap creates a deep copy of a string map
func deepCopyStringMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}

	copy := make(map[string]string, len(m))
	for k, v := range m {
		copy[k] = v
	}
	return copy
}
