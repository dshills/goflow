package transform

import (
	"context"
	"fmt"
	"strings"
)

// TransformType represents the type of transformation to apply
type TransformType int

const (
	TransformTypeUnknown TransformType = iota
	TransformTypeJSONPath
	TransformTypeExpression
	TransformTypeTemplate
)

// Transformer provides a unified interface for all transformation types
type Transformer struct {
	jsonPath   JSONPathQuerier
	expression ExpressionEvaluator
	template   TemplateRenderer
}

// NewTransformer creates a new unified transformer with all capabilities
func NewTransformer() *Transformer {
	return &Transformer{
		jsonPath:   NewJSONPathQuerier(),
		expression: NewExpressionEvaluator(),
		template:   NewTemplateRenderer(),
	}
}

// Transform auto-detects the transformation type and applies it
func (t *Transformer) Transform(ctx context.Context, expr string, data interface{}) (interface{}, error) {
	transformType := detectTransformType(expr)

	switch transformType {
	case TransformTypeJSONPath:
		return t.jsonPath.Query(ctx, expr, data)
	case TransformTypeExpression:
		dataMap, ok := data.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("expression evaluation requires map[string]interface{} context")
		}
		return t.expression.Evaluate(ctx, expr, dataMap)
	case TransformTypeTemplate:
		dataMap, ok := data.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("template rendering requires map[string]interface{} context")
		}
		return t.template.Render(ctx, expr, dataMap)
	default:
		return nil, fmt.Errorf("unable to determine transformation type for expression: %s", expr)
	}
}

// detectTransformType determines which type of transformation should be used
func detectTransformType(expr string) TransformType {
	trimmed := strings.TrimSpace(expr)

	// Check for JSONPath: starts with $ or $.
	if strings.HasPrefix(trimmed, "$.") || trimmed == "$" {
		return TransformTypeJSONPath
	}

	// Check for template: contains ${...} pattern
	if strings.Contains(trimmed, "${") && strings.Contains(trimmed, "}") {
		return TransformTypeTemplate
	}

	// Check for recursive descent
	if strings.Contains(trimmed, "..") {
		return TransformTypeJSONPath
	}

	// Check for filter expressions
	if strings.Contains(trimmed, "[?(") {
		return TransformTypeJSONPath
	}

	// Check for array wildcard
	if strings.Contains(trimmed, "[*]") {
		return TransformTypeJSONPath
	}

	// Check for expression operators
	expressionOperators := []string{
		"==", "!=", ">=", "<=", ">", "<",
		"&&", "||", "!",
		" ? ", " : ", // Ternary operator
		" + ", " - ", " * ", " / ",
	}
	for _, op := range expressionOperators {
		if strings.Contains(trimmed, op) {
			return TransformTypeExpression
		}
	}

	// Default to expression for simple variable access
	return TransformTypeExpression
}

// TransformJSONPath explicitly applies a JSONPath query
func TransformJSONPath(ctx context.Context, path string, data interface{}) (interface{}, error) {
	querier := NewJSONPathQuerier()
	return querier.Query(ctx, path, data)
}

// TransformExpression explicitly evaluates an expression
func TransformExpression(ctx context.Context, expression string, context map[string]interface{}) (interface{}, error) {
	evaluator := NewExpressionEvaluator()
	return evaluator.Evaluate(ctx, expression, context)
}

// TransformTemplate explicitly renders a template
func TransformTemplate(ctx context.Context, template string, context map[string]interface{}) (string, error) {
	renderer := NewTemplateRenderer()
	return renderer.Render(ctx, template, context)
}
