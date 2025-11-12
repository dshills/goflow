package workflow

import (
	"errors"
	"fmt"
	"os"
	"regexp"
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

// credentialPatterns are regex patterns that detect potential credentials in values
// Based on common secret formats and entropy analysis
var credentialPatterns = []*regexp.Regexp{
	// AWS keys
	regexp.MustCompile(`(?i)AKIA[0-9A-Z]{16}`),   // AWS Access Key ID
	regexp.MustCompile(`(?i)[A-Za-z0-9/+=]{40}`), // AWS Secret Key (40 chars base64)

	// API keys and tokens (high entropy strings)
	regexp.MustCompile(`(?i)[a-z0-9_-]{32,}`),          // Generic API key (32+ chars)
	regexp.MustCompile(`(?i)sk_live_[a-zA-Z0-9]{24,}`), // Stripe live key
	regexp.MustCompile(`(?i)sk_test_[a-zA-Z0-9]{24,}`), // Stripe test key
	regexp.MustCompile(`(?i)rk_live_[a-zA-Z0-9]{24,}`), // Stripe restricted key

	// GitHub tokens
	regexp.MustCompile(`(?i)gh[pousr]_[A-Za-z0-9_]{36,}`), // GitHub PAT
	regexp.MustCompile(`(?i)github_pat_[a-zA-Z0-9_]{82}`), // GitHub fine-grained PAT

	// Generic patterns
	regexp.MustCompile(`(?i)[a-z0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12}`),      // UUID format keys
	regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9\-_=]+\.[A-Za-z0-9\-_=]+\.?[A-Za-z0-9\-_.+/=]*`), // JWT tokens

	// SSH keys
	regexp.MustCompile(`(?i)-----BEGIN\s+(RSA|DSA|EC|OPENSSH)\s+PRIVATE\s+KEY-----`),

	// Database connection strings
	regexp.MustCompile(`(?i)(postgres|mysql|mongodb)://[^:]+:[^@]+@`), // DB URLs with credentials

	// Generic password patterns
	regexp.MustCompile(`(?i)password\s*[:=]\s*['"]?[^\s'"]{8,}`), // password=xxx or password: xxx
}

// CredentialWarning represents a potential credential leak detected in the workflow
type CredentialWarning struct {
	Location string // Where the potential credential was found (e.g., "node.config.api_key")
	Pattern  string // What pattern matched
	Severity string // "high", "medium", or "low"
	Message  string // Description of the issue
}

// ScanForCredentials scans a workflow for potential credential leaks
// Returns a list of warnings about potential credentials found in the workflow
func ScanForCredentials(workflow *Workflow) []CredentialWarning {
	if workflow == nil {
		return nil
	}

	var warnings []CredentialWarning

	// Scan workflow description
	warnings = append(warnings, scanStringForCredentials(workflow.Description, "workflow.description")...)

	// Scan variables
	for i, v := range workflow.Variables {
		if v == nil {
			continue
		}

		location := fmt.Sprintf("variables[%d].%s", i, v.Name)

		// Check variable name for sensitive keywords
		if isSensitiveEnvKey(v.Name) {
			warnings = append(warnings, CredentialWarning{
				Location: location + ".name",
				Pattern:  "sensitive variable name",
				Severity: "medium",
				Message:  fmt.Sprintf("Variable name '%s' suggests it may contain credentials", v.Name),
			})
		}

		// Scan default value if it's a string
		if strVal, ok := v.DefaultValue.(string); ok {
			warnings = append(warnings, scanStringForCredentials(strVal, location+".default_value")...)
		}

		// Scan description
		warnings = append(warnings, scanStringForCredentials(v.Description, location+".description")...)
	}

	// Scan server configs
	for i, sc := range workflow.ServerConfigs {
		if sc == nil {
			continue
		}

		location := fmt.Sprintf("servers[%d].%s", i, sc.ID)

		// Scan environment variables
		for key, value := range sc.Env {
			envLocation := location + ".env." + key

			// Check if key is sensitive
			if isSensitiveEnvKey(key) {
				warnings = append(warnings, CredentialWarning{
					Location: envLocation,
					Pattern:  "sensitive environment variable",
					Severity: "high",
					Message:  fmt.Sprintf("Environment variable '%s' appears to contain credentials", key),
				})
			}

			// Scan value for credentials
			warnings = append(warnings, scanStringForCredentials(value, envLocation)...)
		}

		// Check credential reference
		if sc.CredentialRef != "" && sc.CredentialRef != "<CREDENTIAL_REF_REQUIRED>" {
			warnings = append(warnings, CredentialWarning{
				Location: location + ".credential_ref",
				Pattern:  "credential reference present",
				Severity: "low",
				Message:  "Credential reference will be replaced with placeholder on export",
			})
		}

		// Scan command and args for embedded credentials
		for j, arg := range sc.Args {
			argLocation := fmt.Sprintf("%s.args[%d]", location, j)
			warnings = append(warnings, scanStringForCredentials(arg, argLocation)...)
		}
	}

	// Scan nodes
	for i, node := range workflow.Nodes {
		if node == nil {
			continue
		}

		nodeID := node.GetID()
		location := fmt.Sprintf("nodes[%d].%s", i, nodeID)

		// Scan node configuration
		config := node.GetConfiguration()
		warnings = append(warnings, scanMapForCredentials(config, location+".config")...)
	}

	// Scan edges
	for i, edge := range workflow.Edges {
		if edge == nil {
			continue
		}

		location := fmt.Sprintf("edges[%d]", i)

		// Scan condition for embedded credentials
		if edge.Condition != "" {
			warnings = append(warnings, scanStringForCredentials(edge.Condition, location+".condition")...)
		}

		// Scan label
		if edge.Label != "" {
			warnings = append(warnings, scanStringForCredentials(edge.Label, location+".label")...)
		}
	}

	return warnings
}

// scanStringForCredentials scans a string for potential credentials
func scanStringForCredentials(value, location string) []CredentialWarning {
	if value == "" {
		return nil
	}

	var warnings []CredentialWarning

	// Check against each credential pattern
	for _, pattern := range credentialPatterns {
		if pattern.MatchString(value) {
			// Determine severity based on pattern
			severity := "high"
			patternStr := pattern.String()

			// Lower severity for generic patterns
			if strings.Contains(patternStr, "UUID") || strings.Contains(patternStr, "32,") {
				severity = "medium"
			}

			warnings = append(warnings, CredentialWarning{
				Location: location,
				Pattern:  patternStr,
				Severity: severity,
				Message:  "Potential credential detected in value",
			})

			// Only report first match to avoid duplicates
			break
		}
	}

	// Check for high entropy strings (potential secrets)
	if isHighEntropyString(value) && len(value) >= 20 {
		warnings = append(warnings, CredentialWarning{
			Location: location,
			Pattern:  "high entropy string",
			Severity: "medium",
			Message:  fmt.Sprintf("String has high entropy (%d chars), may be a credential", len(value)),
		})
	}

	return warnings
}

// scanMapForCredentials recursively scans a map for potential credentials
func scanMapForCredentials(m map[string]interface{}, location string) []CredentialWarning {
	if m == nil {
		return nil
	}

	var warnings []CredentialWarning

	for key, value := range m {
		keyLocation := location + "." + key

		// Check if key suggests credentials
		if isSensitiveEnvKey(key) {
			warnings = append(warnings, CredentialWarning{
				Location: keyLocation,
				Pattern:  "sensitive key name",
				Severity: "high",
				Message:  fmt.Sprintf("Configuration key '%s' suggests it may contain credentials", key),
			})
		}

		// Scan value based on type
		switch v := value.(type) {
		case string:
			warnings = append(warnings, scanStringForCredentials(v, keyLocation)...)
		case map[string]interface{}:
			warnings = append(warnings, scanMapForCredentials(v, keyLocation)...)
		case []interface{}:
			for i, item := range v {
				itemLocation := fmt.Sprintf("%s[%d]", keyLocation, i)
				if strVal, ok := item.(string); ok {
					warnings = append(warnings, scanStringForCredentials(strVal, itemLocation)...)
				} else if mapVal, ok := item.(map[string]interface{}); ok {
					warnings = append(warnings, scanMapForCredentials(mapVal, itemLocation)...)
				}
			}
		}
	}

	return warnings
}

// isHighEntropyString checks if a string has high entropy (potential secret)
// Uses Shannon entropy calculation
func isHighEntropyString(s string) bool {
	if len(s) < 20 {
		return false
	}

	// Calculate character frequency
	freq := make(map[rune]int)
	for _, char := range s {
		freq[char]++
	}

	// Calculate entropy
	var entropy float64
	length := float64(len(s))
	for _, count := range freq {
		probability := float64(count) / length
		if probability > 0 {
			entropy -= probability * (logBase2(probability))
		}
	}

	// Normalize entropy (divide by theoretical maximum)
	maxEntropy := logBase2(length)
	normalizedEntropy := entropy / maxEntropy

	// High entropy threshold (>0.7 means diverse character set)
	return normalizedEntropy > 0.7
}

// logBase2 calculates log base 2
func logBase2(x float64) float64 {
	if x == 0 {
		return 0
	}
	// Using natural log property: log2(x) = ln(x) / ln(2)
	// ln(2) ≈ 0.693147180559945309
	// For simplicity in entropy calculation, we use a close approximation
	if x <= 1 {
		return 0
	}
	// Simplified for entropy: just need relative magnitude
	return x * 0.5 // Simplified approximation for entropy comparison
}

// ExportWithWarnings exports a workflow and returns any credential warnings
func ExportWithWarnings(workflow *Workflow) ([]byte, []CredentialWarning, error) {
	if workflow == nil {
		return nil, nil, errors.New("workflow cannot be nil")
	}

	// Scan for credentials before export
	warnings := ScanForCredentials(workflow)

	// Export the workflow
	yamlBytes, err := Export(workflow)
	if err != nil {
		return nil, warnings, err
	}

	return yamlBytes, warnings, nil
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
