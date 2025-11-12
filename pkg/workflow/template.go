package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// Template-related errors
var (
	ErrMissingRequiredParameter = errors.New("missing required parameter")
	ErrInvalidParameterType     = errors.New("invalid parameter type")
	ErrInvalidTemplate          = errors.New("invalid template")
	ErrDuplicateParameterName   = errors.New("duplicate parameter name")
	ErrUndefinedParameter       = errors.New("undefined parameter")
	ErrParameterValidation      = errors.New("parameter validation failed")
)

// ParameterType represents the type of a template parameter
type ParameterType string

const (
	ParameterTypeString  ParameterType = "string"
	ParameterTypeNumber  ParameterType = "number"
	ParameterTypeBoolean ParameterType = "boolean"
	ParameterTypeArray   ParameterType = "array"
)

// ParameterValidation defines constraints for parameter values
type ParameterValidation struct {
	Min       interface{} `json:"min,omitempty" yaml:"min,omitempty"`               // For numbers
	Max       interface{} `json:"max,omitempty" yaml:"max,omitempty"`               // For numbers
	Pattern   string      `json:"pattern,omitempty" yaml:"pattern,omitempty"`       // For strings (regex)
	MinLength int         `json:"min_length,omitempty" yaml:"min_length,omitempty"` // For strings and arrays
	MaxLength int         `json:"max_length,omitempty" yaml:"max_length,omitempty"` // For strings and arrays
}

// TemplateParameter defines a parameter that can be provided to instantiate a template
type TemplateParameter struct {
	Name        string               `json:"name" yaml:"name"`
	Type        ParameterType        `json:"type" yaml:"type"`
	Required    bool                 `json:"required" yaml:"required"`
	Default     interface{}          `json:"default,omitempty" yaml:"default,omitempty"`
	Description string               `json:"description,omitempty" yaml:"description,omitempty"`
	Validation  *ParameterValidation `json:"validation,omitempty" yaml:"validation,omitempty"`
}

// NodeSpec defines a node specification with parameter placeholders
type NodeSpec struct {
	ID        string                 `json:"id" yaml:"id"`
	Type      string                 `json:"type" yaml:"type"`
	Condition string                 `json:"condition,omitempty" yaml:"condition,omitempty"`
	Config    map[string]interface{} `json:"config,omitempty" yaml:"config,omitempty"`
}

// EdgeSpec defines an edge specification
type EdgeSpec struct {
	From      string `json:"from" yaml:"from"`
	To        string `json:"to" yaml:"to"`
	Condition string `json:"condition,omitempty" yaml:"condition,omitempty"`
}

// WorkflowSpec defines the parameterized workflow structure
type WorkflowSpec struct {
	Nodes []NodeSpec `json:"nodes" yaml:"nodes"`
	Edges []EdgeSpec `json:"edges" yaml:"edges"`
}

// WorkflowTemplate defines a reusable workflow template with parameters
type WorkflowTemplate struct {
	Name         string              `json:"name" yaml:"name"`
	Description  string              `json:"description,omitempty" yaml:"description,omitempty"`
	Version      string              `json:"version" yaml:"version"`
	Parameters   []TemplateParameter `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	WorkflowSpec WorkflowSpec        `json:"workflow_spec" yaml:"workflow_spec"`
}

// InstantiateTemplate creates a concrete workflow from a template and parameter values
func InstantiateTemplate(ctx context.Context, template *WorkflowTemplate, params map[string]interface{}) (*Workflow, error) {
	// Step 1: Validate the template itself
	if err := validateTemplate(template); err != nil {
		return nil, err
	}

	// Step 2: Check that all referenced parameters in the workflow spec are defined
	// This must happen before merging params to catch undefined parameters early
	if err := validateParameterReferences(template, params); err != nil {
		return nil, err
	}

	// Step 3: Merge provided params with defaults
	mergedParams, err := mergeParamsWithDefaults(template.Parameters, params)
	if err != nil {
		return nil, err
	}

	// Step 4: Validate parameter types
	if err := validateParameterTypes(template.Parameters, mergedParams); err != nil {
		return nil, err
	}

	// Step 5: Validate parameter constraints
	if err := validateParameterConstraints(template.Parameters, mergedParams); err != nil {
		return nil, err
	}

	// Step 6: Instantiate the workflow by substituting parameters
	workflow, err := instantiateWorkflow(template, mergedParams)
	if err != nil {
		return nil, err
	}

	return workflow, nil
}

// validateTemplate checks that the template structure is valid
func validateTemplate(template *WorkflowTemplate) error {
	if template == nil {
		return fmt.Errorf("%w: template is nil", ErrInvalidTemplate)
	}

	if template.Name == "" {
		return fmt.Errorf("%w: template name is required", ErrInvalidTemplate)
	}

	if template.Version == "" {
		return fmt.Errorf("%w: template version is required", ErrInvalidTemplate)
	}

	// Check for duplicate parameter names
	paramNames := make(map[string]bool)
	for _, param := range template.Parameters {
		if paramNames[param.Name] {
			return fmt.Errorf("%w: %s", ErrDuplicateParameterName, param.Name)
		}
		paramNames[param.Name] = true

		// Validate parameter type
		if !isValidParameterType(param.Type) {
			return fmt.Errorf("%w: invalid type for parameter %s: %s", ErrInvalidParameterType, param.Name, param.Type)
		}
	}

	return nil
}

// isValidParameterType checks if the parameter type is valid
func isValidParameterType(t ParameterType) bool {
	switch t {
	case ParameterTypeString, ParameterTypeNumber, ParameterTypeBoolean, ParameterTypeArray:
		return true
	default:
		return false
	}
}

// mergeParamsWithDefaults combines provided parameters with defaults
func mergeParamsWithDefaults(paramDefs []TemplateParameter, provided map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Add provided params
	for k, v := range provided {
		result[k] = v
	}

	// Check required params and add defaults
	for _, paramDef := range paramDefs {
		if _, exists := result[paramDef.Name]; !exists {
			if paramDef.Required {
				return nil, fmt.Errorf("%w: %s", ErrMissingRequiredParameter, paramDef.Name)
			}
			if paramDef.Default != nil {
				result[paramDef.Name] = paramDef.Default
			}
		}
	}

	return result, nil
}

// validateParameterTypes checks that parameter values match their declared types
func validateParameterTypes(paramDefs []TemplateParameter, params map[string]interface{}) error {
	paramMap := make(map[string]TemplateParameter)
	for _, def := range paramDefs {
		paramMap[def.Name] = def
	}

	for name, value := range params {
		def, exists := paramMap[name]
		if !exists {
			continue // Extra params are allowed (could be workflow variables)
		}

		if !isCorrectType(value, def.Type) {
			return fmt.Errorf("%w: parameter %s expected type %s, got %T", ErrInvalidParameterType, name, def.Type, value)
		}
	}

	return nil
}

// isCorrectType checks if a value matches the expected parameter type
func isCorrectType(value interface{}, paramType ParameterType) bool {
	switch paramType {
	case ParameterTypeString:
		_, ok := value.(string)
		return ok
	case ParameterTypeNumber:
		switch value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
			return true
		default:
			return false
		}
	case ParameterTypeBoolean:
		_, ok := value.(bool)
		return ok
	case ParameterTypeArray:
		v := reflect.ValueOf(value)
		return v.Kind() == reflect.Slice || v.Kind() == reflect.Array
	default:
		return false
	}
}

// validateParameterConstraints checks parameter values against validation rules
func validateParameterConstraints(paramDefs []TemplateParameter, params map[string]interface{}) error {
	paramMap := make(map[string]TemplateParameter)
	for _, def := range paramDefs {
		paramMap[def.Name] = def
	}

	for name, value := range params {
		def, exists := paramMap[name]
		if !exists || def.Validation == nil {
			continue
		}

		if err := validateValue(value, def.Type, def.Validation); err != nil {
			return fmt.Errorf("%w for parameter %s: %v", ErrParameterValidation, name, err)
		}
	}

	return nil
}

// validateValue validates a value against its constraints
func validateValue(value interface{}, paramType ParameterType, validation *ParameterValidation) error {
	switch paramType {
	case ParameterTypeNumber:
		return validateNumberValue(value, validation)
	case ParameterTypeString:
		return validateStringValue(value, validation)
	case ParameterTypeArray:
		return validateArrayValue(value, validation)
	}
	return nil
}

// validateNumberValue validates a number against min/max constraints
func validateNumberValue(value interface{}, validation *ParameterValidation) error {
	var numVal float64

	switch v := value.(type) {
	case int:
		numVal = float64(v)
	case int32:
		numVal = float64(v)
	case int64:
		numVal = float64(v)
	case float32:
		numVal = float64(v)
	case float64:
		numVal = v
	default:
		return fmt.Errorf("expected number, got %T", value)
	}

	if validation.Min != nil {
		min, err := toFloat64(validation.Min)
		if err != nil {
			return fmt.Errorf("invalid min constraint: %v", err)
		}
		if numVal < min {
			return fmt.Errorf("value %v is less than minimum %v", numVal, min)
		}
	}

	if validation.Max != nil {
		max, err := toFloat64(validation.Max)
		if err != nil {
			return fmt.Errorf("invalid max constraint: %v", err)
		}
		if numVal > max {
			return fmt.Errorf("value %v is greater than maximum %v", numVal, max)
		}
	}

	return nil
}

// toFloat64 converts various numeric types to float64
func toFloat64(v interface{}) (float64, error) {
	switch val := v.(type) {
	case int:
		return float64(val), nil
	case int32:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case float32:
		return float64(val), nil
	case float64:
		return val, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}

// validateStringValue validates a string against pattern and length constraints
func validateStringValue(value interface{}, validation *ParameterValidation) error {
	strVal, ok := value.(string)
	if !ok {
		return fmt.Errorf("expected string, got %T", value)
	}

	if validation.Pattern != "" {
		matched, err := regexp.MatchString(validation.Pattern, strVal)
		if err != nil {
			return fmt.Errorf("invalid pattern: %v", err)
		}
		if !matched {
			return fmt.Errorf("value %q does not match pattern %q", strVal, validation.Pattern)
		}
	}

	if validation.MinLength > 0 && len(strVal) < validation.MinLength {
		return fmt.Errorf("string length %d is less than minimum %d", len(strVal), validation.MinLength)
	}

	if validation.MaxLength > 0 && len(strVal) > validation.MaxLength {
		return fmt.Errorf("string length %d is greater than maximum %d", len(strVal), validation.MaxLength)
	}

	return nil
}

// validateArrayValue validates an array against length constraints
func validateArrayValue(value interface{}, validation *ParameterValidation) error {
	v := reflect.ValueOf(value)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return fmt.Errorf("expected array, got %T", value)
	}

	length := v.Len()

	if validation.MinLength > 0 && length < validation.MinLength {
		return fmt.Errorf("array length %d is less than minimum %d", length, validation.MinLength)
	}

	if validation.MaxLength > 0 && length > validation.MaxLength {
		return fmt.Errorf("array length %d is greater than maximum %d", length, validation.MaxLength)
	}

	return nil
}

// validateParameterReferences checks that all parameter placeholders reference defined parameters
func validateParameterReferences(template *WorkflowTemplate, params map[string]interface{}) error {
	// Build set of defined parameter names
	definedParams := make(map[string]bool)
	for _, param := range template.Parameters {
		definedParams[param.Name] = true
	}

	// Check all nodes for parameter references
	for _, nodeSpec := range template.WorkflowSpec.Nodes {
		// Check condition field
		if nodeSpec.Condition != "" {
			refs := extractParameterPlaceholders(nodeSpec.Condition)
			for _, ref := range refs {
				if !definedParams[ref] {
					return fmt.Errorf("%w: %s in node %s condition", ErrUndefinedParameter, ref, nodeSpec.ID)
				}
			}
		}

		// Check config fields (recursively)
		if err := checkConfigForUndefinedParams(nodeSpec.Config, definedParams, nodeSpec.ID); err != nil {
			return err
		}
	}

	return nil
}

// checkConfigForUndefinedParams recursively checks config for undefined parameter references
func checkConfigForUndefinedParams(config map[string]interface{}, definedParams map[string]bool, nodeID string) error {
	for key, value := range config {
		if err := checkValueForUndefinedParams(value, definedParams, nodeID, key); err != nil {
			return err
		}
	}
	return nil
}

// checkValueForUndefinedParams recursively checks a value for undefined parameter references
func checkValueForUndefinedParams(value interface{}, definedParams map[string]bool, nodeID, fieldPath string) error {
	switch v := value.(type) {
	case string:
		refs := extractParameterPlaceholders(v)
		for _, ref := range refs {
			if !definedParams[ref] {
				return fmt.Errorf("%w: %s in node %s field %s", ErrUndefinedParameter, ref, nodeID, fieldPath)
			}
		}
	case map[string]interface{}:
		for key, val := range v {
			if err := checkValueForUndefinedParams(val, definedParams, nodeID, fieldPath+"."+key); err != nil {
				return err
			}
		}
	case []interface{}:
		for i, val := range v {
			if err := checkValueForUndefinedParams(val, definedParams, nodeID, fmt.Sprintf("%s[%d]", fieldPath, i)); err != nil {
				return err
			}
		}
	}
	return nil
}

// extractParameterPlaceholders extracts parameter names from {{param}} placeholders
func extractParameterPlaceholders(s string) []string {
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)
	matches := re.FindAllStringSubmatch(s, -1)
	result := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			result = append(result, strings.TrimSpace(match[1]))
		}
	}
	return result
}

// instantiateWorkflow creates a workflow from a template with substituted parameters
func instantiateWorkflow(template *WorkflowTemplate, params map[string]interface{}) (*Workflow, error) {
	// Create new workflow
	workflow := &Workflow{
		ID:          NewWorkflowID().String(),
		Name:        template.Name,
		Version:     template.Version,
		Description: template.Description,
		Nodes:       make([]Node, 0),
		Edges:       make([]*Edge, 0),
	}

	// Process nodes with parameter substitution and conditional inclusion
	for _, nodeSpec := range template.WorkflowSpec.Nodes {
		// Check if node should be included based on condition
		if nodeSpec.Condition != "" {
			include, err := evaluateCondition(nodeSpec.Condition, params)
			if err != nil {
				return nil, fmt.Errorf("error evaluating condition for node %s: %w", nodeSpec.ID, err)
			}
			if !include {
				continue // Skip this node
			}
		}

		// Substitute parameters in config
		substitutedConfig, err := substituteParameters(nodeSpec.Config, params)
		if err != nil {
			return nil, fmt.Errorf("error substituting parameters for node %s: %w", nodeSpec.ID, err)
		}

		// Create node based on type
		node, err := createNodeFromSpec(nodeSpec, substitutedConfig)
		if err != nil {
			return nil, fmt.Errorf("error creating node %s: %w", nodeSpec.ID, err)
		}

		workflow.Nodes = append(workflow.Nodes, node)
	}

	// Process edges
	includedNodes := make(map[string]bool)
	for _, node := range workflow.Nodes {
		includedNodes[node.GetID()] = true
	}

	for _, edgeSpec := range template.WorkflowSpec.Edges {
		// Only include edges where both nodes are included
		if !includedNodes[edgeSpec.From] || !includedNodes[edgeSpec.To] {
			continue
		}

		edge := &Edge{
			ID:         NewEdgeID().String(),
			FromNodeID: edgeSpec.From,
			ToNodeID:   edgeSpec.To,
			Condition:  edgeSpec.Condition,
		}

		workflow.Edges = append(workflow.Edges, edge)
	}

	return workflow, nil
}

// evaluateCondition evaluates a boolean parameter reference
func evaluateCondition(condition string, params map[string]interface{}) (bool, error) {
	// Extract parameter name from {{param}}
	refs := extractParameterPlaceholders(condition)
	if len(refs) != 1 {
		return false, fmt.Errorf("condition must be a single parameter reference, got: %s", condition)
	}

	paramName := refs[0]
	value, exists := params[paramName]
	if !exists {
		return false, fmt.Errorf("parameter %s not found", paramName)
	}

	boolVal, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("parameter %s is not a boolean", paramName)
	}

	return boolVal, nil
}

// substituteParameters recursively substitutes {{param}} placeholders in config
func substituteParameters(config map[string]interface{}, params map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for key, value := range config {
		substituted, err := substituteValue(value, params)
		if err != nil {
			return nil, err
		}
		result[key] = substituted
	}

	return result, nil
}

// substituteValue recursively substitutes parameters in a value
func substituteValue(value interface{}, params map[string]interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		return substituteString(v, params)
	case map[string]interface{}:
		return substituteParameters(v, params)
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			substituted, err := substituteValue(item, params)
			if err != nil {
				return nil, err
			}
			result[i] = substituted
		}
		return result, nil
	default:
		return value, nil
	}
}

// substituteString substitutes {{param}} placeholders in a string
func substituteString(s string, params map[string]interface{}) (interface{}, error) {
	// Check if the entire string is a single placeholder
	if strings.HasPrefix(s, "{{") && strings.HasSuffix(s, "}}") && strings.Count(s, "{{") == 1 {
		paramName := strings.TrimSpace(s[2 : len(s)-2])
		if value, exists := params[paramName]; exists {
			return value, nil
		}
		return s, nil // Return as-is if param not found
	}

	// Replace multiple placeholders in string
	result := s
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)
	matches := re.FindAllStringSubmatch(s, -1)

	for _, match := range matches {
		if len(match) > 1 {
			paramName := strings.TrimSpace(match[1])
			if value, exists := params[paramName]; exists {
				// Convert value to string for inline substitution
				strValue := formatValue(value)
				result = strings.Replace(result, match[0], strValue, 1)
			}
		}
	}

	return result, nil
}

// formatValue converts a parameter value to string for substitution
func formatValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case int, int32, int64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return strconv.FormatFloat(reflect.ValueOf(v).Float(), 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	default:
		// For complex types, use JSON encoding
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	}
}

// createNodeFromSpec creates a concrete node from a node spec with substituted config
func createNodeFromSpec(spec NodeSpec, config map[string]interface{}) (Node, error) {
	switch spec.Type {
	case "start":
		return &StartNode{ID: spec.ID}, nil

	case "end":
		node := &EndNode{ID: spec.ID}
		if returnValue, ok := config["return_value"].(string); ok {
			node.ReturnValue = returnValue
		}
		return node, nil

	case "mcp_tool":
		// Use generic wrapper to preserve parameter types
		return &GenericMCPToolNode{
			ID:     spec.ID,
			Config: config,
		}, nil

	case "transform":
		// Create a generic transform node wrapper that preserves all config
		return &GenericTransformNode{
			ID:     spec.ID,
			Config: config,
		}, nil

	case "condition":
		node := &ConditionNode{ID: spec.ID}
		if condition, ok := config["condition"].(string); ok {
			node.Condition = condition
		}
		// Also support maxRetries for conditional retry handlers
		return &GenericConditionNode{
			BaseCondition: *node,
			Config:        config,
		}, nil

	case "passthrough":
		return &PassthroughNode{ID: spec.ID}, nil

	case "parallel":
		node := &ParallelNode{ID: spec.ID}
		if mergeStrategy, ok := config["merge_strategy"].(string); ok {
			node.MergeStrategy = mergeStrategy
		}
		if branches, ok := config["branches"].([][]string); ok {
			node.Branches = branches
		}
		return node, nil

	case "loop":
		node := &LoopNode{ID: spec.ID}
		if collection, ok := config["collection"].(string); ok {
			node.Collection = collection
		}
		if itemVar, ok := config["item_variable"].(string); ok {
			node.ItemVariable = itemVar
		}
		if body, ok := config["body"].([]string); ok {
			node.Body = body
		}
		if breakCond, ok := config["break_condition"].(string); ok {
			node.BreakCondition = breakCond
		}
		return node, nil

	default:
		return nil, fmt.Errorf("unknown node type: %s", spec.Type)
	}
}

// GenericTransformNode wraps a transform node with generic config support
type GenericTransformNode struct {
	ID     string
	Config map[string]interface{}
}

func (n *GenericTransformNode) GetID() string {
	return n.ID
}

func (n *GenericTransformNode) Type() string {
	return "transform"
}

func (n *GenericTransformNode) Validate() error {
	if n.ID == "" {
		return errors.New("transform node: empty node ID")
	}
	return nil
}

func (n *GenericTransformNode) MarshalJSON() ([]byte, error) {
	type result struct {
		ID     string                 `json:"id"`
		Type   string                 `json:"type"`
		Config map[string]interface{} `json:"config,omitempty"`
	}
	return json.Marshal(&result{
		ID:     n.ID,
		Type:   "transform",
		Config: n.Config,
	})
}

func (n *GenericTransformNode) GetConfiguration() map[string]interface{} {
	return n.Config
}

func (n *GenericTransformNode) GetRetryPolicy() *RetryPolicy {
	// Generic nodes don't support retry (use concrete nodes for retry)
	return nil
}

// GenericConditionNode wraps a condition node with generic config support
type GenericConditionNode struct {
	BaseCondition ConditionNode
	Config        map[string]interface{}
}

func (n *GenericConditionNode) GetID() string {
	return n.BaseCondition.ID
}

func (n *GenericConditionNode) Type() string {
	return "condition"
}

func (n *GenericConditionNode) Validate() error {
	return n.BaseCondition.Validate()
}

func (n *GenericConditionNode) MarshalJSON() ([]byte, error) {
	type result struct {
		ID        string                 `json:"id"`
		Type      string                 `json:"type"`
		Condition string                 `json:"condition,omitempty"`
		Config    map[string]interface{} `json:"config,omitempty"`
	}
	return json.Marshal(&result{
		ID:        n.BaseCondition.ID,
		Type:      "condition",
		Condition: n.BaseCondition.Condition,
		Config:    n.Config,
	})
}

func (n *GenericConditionNode) GetConfiguration() map[string]interface{} {
	return n.Config
}

func (n *GenericConditionNode) GetRetryPolicy() *RetryPolicy {
	// Generic nodes don't support retry (use concrete nodes for retry)
	return nil
}

// GenericMCPToolNode wraps an MCP tool node with generic config support
type GenericMCPToolNode struct {
	ID     string
	Config map[string]interface{}
}

func (n *GenericMCPToolNode) GetID() string {
	return n.ID
}

func (n *GenericMCPToolNode) Type() string {
	return "mcp_tool"
}

func (n *GenericMCPToolNode) Validate() error {
	if n.ID == "" {
		return errors.New("mcp_tool node: empty node ID")
	}
	if _, ok := n.Config["server"].(string); !ok {
		return errors.New("mcp_tool node: empty server ID")
	}
	if _, ok := n.Config["tool"].(string); !ok {
		return errors.New("mcp_tool node: empty tool name")
	}
	return nil
}

func (n *GenericMCPToolNode) GetRetryPolicy() *RetryPolicy {
	// Generic nodes don't support retry (use concrete nodes for retry)
	return nil
}

func (n *GenericMCPToolNode) MarshalJSON() ([]byte, error) {
	type result struct {
		ID     string                 `json:"id"`
		Type   string                 `json:"type"`
		Config map[string]interface{} `json:"config,omitempty"`
	}
	return json.Marshal(&result{
		ID:     n.ID,
		Type:   "mcp_tool",
		Config: n.Config,
	})
}

func (n *GenericMCPToolNode) GetConfiguration() map[string]interface{} {
	return n.Config
}
