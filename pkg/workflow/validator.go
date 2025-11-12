package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/xeipuuv/gojsonschema"
)

// ValidateAgainstSchema validates workflow YAML bytes against the JSON schema
func ValidateAgainstSchema(yamlBytes []byte) error {
	if len(yamlBytes) == 0 {
		return errors.New("empty YAML input")
	}

	// Parse YAML into a generic structure that can be validated
	// gojsonschema can work with Go data structures
	var data interface{}

	// Try to parse as YAML first (since that's what we expect)
	wf, err := Parse(yamlBytes)
	if err != nil {
		return fmt.Errorf("failed to parse YAML for validation: %w", err)
	}

	// Convert workflow to JSON for validation (schema validator works with JSON)
	jsonBytes, err := json.Marshal(wf)
	if err != nil {
		return fmt.Errorf("failed to convert workflow to JSON for validation: %w", err)
	}

	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return fmt.Errorf("failed to unmarshal workflow JSON: %w", err)
	}

	// Load the schema
	schemaPath := "specs/001-goflow-spec-review/contracts/workflow-schema-v1.json"
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	// Create schema loader
	schemaLoader := gojsonschema.NewBytesLoader(schemaBytes)
	documentLoader := gojsonschema.NewGoLoader(data)

	// Validate
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("schema validation error: %w", err)
	}

	if !result.Valid() {
		// Collect all validation errors
		var errMsg string
		for i, desc := range result.Errors() {
			if i > 0 {
				errMsg += "; "
			}
			errMsg += fmt.Sprintf("%s: %s", desc.Field(), desc.Description())
		}
		return fmt.Errorf("schema validation failed: %s", errMsg)
	}

	return nil
}

// TopologicalSort performs a topological sort on the workflow nodes
// Returns an ordered list of node IDs that respects the dependency order
func TopologicalSort(workflow *Workflow) ([]NodeID, error) {
	if workflow == nil {
		return nil, errors.New("workflow cannot be nil")
	}

	// Build adjacency list and in-degree map
	adjacency := make(map[string][]string)
	inDegree := make(map[string]int)

	// Initialize in-degree for all nodes
	for _, node := range workflow.Nodes {
		nodeID := node.GetID()
		inDegree[nodeID] = 0
		adjacency[nodeID] = []string{}
	}

	// Build adjacency list and calculate in-degrees
	for _, edge := range workflow.Edges {
		adjacency[edge.FromNodeID] = append(adjacency[edge.FromNodeID], edge.ToNodeID)
		inDegree[edge.ToNodeID]++
	}

	// Kahn's algorithm: start with nodes that have no incoming edges
	queue := make([]string, 0)
	for _, node := range workflow.Nodes {
		nodeID := node.GetID()
		if inDegree[nodeID] == 0 {
			queue = append(queue, nodeID)
		}
	}

	// Process nodes in topological order
	result := make([]NodeID, 0, len(workflow.Nodes))
	for len(queue) > 0 {
		// Remove node from queue
		current := queue[0]
		queue = queue[1:]
		result = append(result, NodeID(current))

		// For each neighbor, reduce in-degree
		for _, neighbor := range adjacency[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// If we haven't processed all nodes, there's a cycle
	if len(result) != len(workflow.Nodes) {
		return nil, errors.New("workflow contains a cycle (circular dependency)")
	}

	return result, nil
}

// Input Validation Functions for Security Hardening (T192)

const (
	maxWorkflowNameLength = 256
	maxNodeIDLength       = 128
	maxVariableNameLength = 128
	maxPathLength         = 4096
	maxExpressionLength   = 8192
	maxDescriptionLength  = 4096
	maxTagLength          = 64
	maxTags               = 50
)

var (
	// validIdentifierRegex matches valid identifiers (alphanumeric, underscore, hyphen)
	validIdentifierRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

	// validVersionRegex matches semantic versioning
	validVersionRegex = regexp.MustCompile(`^v?\d+\.\d+\.\d+(-[a-zA-Z0-9.-]+)?(\+[a-zA-Z0-9.-]+)?$`)

	// pathTraversalPatterns detects path traversal attempts
	pathTraversalPatterns = []string{
		"..",
		"./",
		"../",
		"..\\",
		".\\",
		"%2e%2e",
		"%252e%252e",
		"..%2f",
		"..%5c",
	}

	// suspiciousPatterns detects potential malicious content (all lowercase for case-insensitive matching)
	suspiciousPatterns = []string{
		"<script",
		"javascript:",
		"data:text/html",
		"vbscript:",
		"onload=",
		"onerror=",
		"onclick=",
		"eval(",
		"exec(",
		"system(",
		"popen(",
		"subprocess.",
		"os.system",
		"__import__",
		"ld_preload",
		"ld_library_path",
	}

	// nullBytePatterns detects null byte injection
	nullBytePatterns = []string{
		"\x00",
		"%00",
		"\\0",
		"\\x00",
		"\\u0000",
	}
)

// ValidateWorkflowName validates a workflow name for security
func ValidateWorkflowName(name string) error {
	if name == "" {
		return errors.New("workflow name cannot be empty")
	}

	if len(name) > maxWorkflowNameLength {
		return fmt.Errorf("workflow name exceeds maximum length of %d characters", maxWorkflowNameLength)
	}

	if !utf8.ValidString(name) {
		return errors.New("workflow name contains invalid UTF-8 characters")
	}

	// Check for null bytes
	if err := validateNoNullBytes(name, "workflow name"); err != nil {
		return err
	}

	// Check for control characters
	if err := validateNoControlChars(name, "workflow name"); err != nil {
		return err
	}

	// Check for path traversal patterns
	if err := validateNoPathTraversal(name, "workflow name"); err != nil {
		return err
	}

	return nil
}

// ValidateNodeID validates a node ID for security
func ValidateNodeID(id string) error {
	if id == "" {
		return errors.New("node ID cannot be empty")
	}

	if len(id) > maxNodeIDLength {
		return fmt.Errorf("node ID exceeds maximum length of %d characters", maxNodeIDLength)
	}

	if !utf8.ValidString(id) {
		return errors.New("node ID contains invalid UTF-8 characters")
	}

	// Node IDs should be valid identifiers
	if !validIdentifierRegex.MatchString(id) {
		return fmt.Errorf("node ID must start with a letter and contain only alphanumeric characters, underscores, or hyphens")
	}

	// Check for null bytes
	if err := validateNoNullBytes(id, "node ID"); err != nil {
		return err
	}

	return nil
}

// ValidateVariableName validates a variable name for security
func ValidateVariableName(name string) error {
	if name == "" {
		return errors.New("variable name cannot be empty")
	}

	if len(name) > maxVariableNameLength {
		return fmt.Errorf("variable name exceeds maximum length of %d characters", maxVariableNameLength)
	}

	if !utf8.ValidString(name) {
		return errors.New("variable name contains invalid UTF-8 characters")
	}

	// Variable names should be valid identifiers
	if !validIdentifierRegex.MatchString(name) {
		return fmt.Errorf("variable name must start with a letter and contain only alphanumeric characters, underscores, or hyphens")
	}

	// Check for null bytes
	if err := validateNoNullBytes(name, "variable name"); err != nil {
		return err
	}

	// Check for reserved words
	if isReservedWord(name) {
		return fmt.Errorf("variable name '%s' is a reserved word", name)
	}

	return nil
}

// ValidateExpression validates an expression for security
func ValidateExpression(expr string) error {
	if expr == "" {
		return errors.New("expression cannot be empty")
	}

	if len(expr) > maxExpressionLength {
		return fmt.Errorf("expression exceeds maximum length of %d characters", maxExpressionLength)
	}

	if !utf8.ValidString(expr) {
		return errors.New("expression contains invalid UTF-8 characters")
	}

	// Check for null bytes
	if err := validateNoNullBytes(expr, "expression"); err != nil {
		return err
	}

	// Check for suspicious patterns
	if err := validateNoSuspiciousPatterns(expr, "expression"); err != nil {
		return err
	}

	// Additional expression-specific validation is done by validateExpressionSyntax
	return validateExpressionSyntax(expr)
}

// ValidateFilePath validates a file path for security
func ValidateFilePath(path string) error {
	if path == "" {
		return errors.New("file path cannot be empty")
	}

	if len(path) > maxPathLength {
		return fmt.Errorf("file path exceeds maximum length of %d characters", maxPathLength)
	}

	if !utf8.ValidString(path) {
		return errors.New("file path contains invalid UTF-8 characters")
	}

	// Check for null bytes
	if err := validateNoNullBytes(path, "file path"); err != nil {
		return err
	}

	// Check for path traversal
	if err := validateNoPathTraversal(path, "file path"); err != nil {
		return err
	}

	// Clean and validate the path
	cleaned := filepath.Clean(path)
	if strings.HasPrefix(cleaned, "..") {
		return errors.New("file path attempts to access parent directories")
	}

	return nil
}

// ValidateVersion validates a version string
func ValidateVersion(version string) error {
	if version == "" {
		return errors.New("version cannot be empty")
	}

	if len(version) > 64 {
		return errors.New("version string too long")
	}

	if !utf8.ValidString(version) {
		return errors.New("version contains invalid UTF-8 characters")
	}

	if !validVersionRegex.MatchString(version) {
		return fmt.Errorf("version must follow semantic versioning format (e.g., 1.0.0)")
	}

	return nil
}

// ValidateDescription validates a description field
func ValidateDescription(desc string) error {
	if len(desc) > maxDescriptionLength {
		return fmt.Errorf("description exceeds maximum length of %d characters", maxDescriptionLength)
	}

	if !utf8.ValidString(desc) {
		return errors.New("description contains invalid UTF-8 characters")
	}

	// Check for null bytes
	if err := validateNoNullBytes(desc, "description"); err != nil {
		return err
	}

	// Check for suspicious patterns
	if err := validateNoSuspiciousPatterns(desc, "description"); err != nil {
		return err
	}

	return nil
}

// ValidateTags validates workflow tags
func ValidateTags(tags []string) error {
	if len(tags) > maxTags {
		return fmt.Errorf("number of tags exceeds maximum of %d", maxTags)
	}

	for i, tag := range tags {
		if tag == "" {
			return fmt.Errorf("tag at index %d is empty", i)
		}

		if len(tag) > maxTagLength {
			return fmt.Errorf("tag '%s' exceeds maximum length of %d characters", tag, maxTagLength)
		}

		if !utf8.ValidString(tag) {
			return fmt.Errorf("tag '%s' contains invalid UTF-8 characters", tag)
		}

		// Check for null bytes
		if err := validateNoNullBytes(tag, fmt.Sprintf("tag '%s'", tag)); err != nil {
			return err
		}

		// Tags should be simple identifiers or phrases
		if strings.ContainsAny(tag, "<>\x00") {
			return fmt.Errorf("tag '%s' contains invalid characters", tag)
		}
	}

	return nil
}

// Helper validation functions

func validateNoNullBytes(s, fieldName string) error {
	for _, pattern := range nullBytePatterns {
		if strings.Contains(s, pattern) {
			return fmt.Errorf("%s contains null byte (potential injection attack)", fieldName)
		}
	}
	return nil
}

func validateNoControlChars(s, fieldName string) error {
	for _, r := range s {
		if unicode.IsControl(r) && !unicode.IsSpace(r) {
			return fmt.Errorf("%s contains control characters", fieldName)
		}
	}
	return nil
}

func validateNoPathTraversal(s, fieldName string) error {
	lowerStr := strings.ToLower(s)
	for _, pattern := range pathTraversalPatterns {
		if strings.Contains(lowerStr, pattern) {
			return fmt.Errorf("%s contains path traversal pattern: %s", fieldName, pattern)
		}
	}
	return nil
}

func validateNoSuspiciousPatterns(s, fieldName string) error {
	lowerStr := strings.ToLower(s)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(lowerStr, pattern) {
			return fmt.Errorf("%s contains suspicious pattern: %s (potential code injection)", fieldName, pattern)
		}
	}
	return nil
}

func isReservedWord(name string) bool {
	reserved := map[string]bool{
		"true":     true,
		"false":    true,
		"nil":      true,
		"null":     true,
		"and":      true,
		"or":       true,
		"not":      true,
		"if":       true,
		"else":     true,
		"then":     true,
		"for":      true,
		"while":    true,
		"break":    true,
		"continue": true,
		"return":   true,
		"function": true,
		"var":      true,
		"let":      true,
		"const":    true,
	}
	return reserved[strings.ToLower(name)]
}

// ValidateWorkflow performs comprehensive validation of a workflow
func ValidateWorkflow(wf *Workflow) error {
	if wf == nil {
		return errors.New("workflow cannot be nil")
	}

	// Validate workflow name
	if err := ValidateWorkflowName(wf.Name); err != nil {
		return fmt.Errorf("invalid workflow name: %w", err)
	}

	// Validate version
	if err := ValidateVersion(wf.Version); err != nil {
		return fmt.Errorf("invalid version: %w", err)
	}

	// Validate description
	if err := ValidateDescription(wf.Description); err != nil {
		return fmt.Errorf("invalid description: %w", err)
	}

	// Validate metadata
	if err := ValidateTags(wf.Metadata.Tags); err != nil {
		return fmt.Errorf("invalid tags: %w", err)
	}

	// Validate variables
	varNames := make(map[string]bool)
	for i, v := range wf.Variables {
		if v == nil {
			return fmt.Errorf("variable at index %d is nil", i)
		}

		if err := ValidateVariableName(v.Name); err != nil {
			return fmt.Errorf("invalid variable at index %d: %w", i, err)
		}

		if varNames[v.Name] {
			return fmt.Errorf("duplicate variable name: %s", v.Name)
		}
		varNames[v.Name] = true

		if err := ValidateDescription(v.Description); err != nil {
			return fmt.Errorf("invalid description for variable '%s': %w", v.Name, err)
		}
	}

	// Validate nodes
	nodeIDs := make(map[string]bool)
	for i, node := range wf.Nodes {
		if node == nil {
			return fmt.Errorf("node at index %d is nil", i)
		}

		nodeID := node.GetID()
		if err := ValidateNodeID(nodeID); err != nil {
			return fmt.Errorf("invalid node at index %d: %w", i, err)
		}

		if nodeIDs[nodeID] {
			return fmt.Errorf("duplicate node ID: %s", nodeID)
		}
		nodeIDs[nodeID] = true

		// Validate node-specific fields
		if err := node.Validate(); err != nil {
			return fmt.Errorf("validation failed for node '%s': %w", nodeID, err)
		}
	}

	// Validate edges
	for i, edge := range wf.Edges {
		if edge == nil {
			return fmt.Errorf("edge at index %d is nil", i)
		}

		if !nodeIDs[edge.FromNodeID] {
			return fmt.Errorf("edge %d references non-existent from_node: %s", i, edge.FromNodeID)
		}

		if !nodeIDs[edge.ToNodeID] {
			return fmt.Errorf("edge %d references non-existent to_node: %s", i, edge.ToNodeID)
		}

		if edge.Condition != "" {
			if err := ValidateExpression(edge.Condition); err != nil {
				return fmt.Errorf("invalid condition in edge %d: %w", i, err)
			}
		}
	}

	// Check for cycles
	if _, err := TopologicalSort(wf); err != nil {
		return fmt.Errorf("workflow validation failed: %w", err)
	}

	return nil
}
