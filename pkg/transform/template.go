package transform

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Sentinel errors for template operations
var (
	ErrInvalidTemplate = errors.New("invalid template syntax")
	ErrUnknownFunction = errors.New("unknown template function")
	ErrNilContext      = errors.New("nil template context")
	ErrInvalidEscape   = errors.New("invalid escape sequence")
)

// TemplateRenderer defines the interface for rendering templates
type TemplateRenderer interface {
	Render(ctx context.Context, template string, context map[string]interface{}) (string, error)
}

// customTemplateRenderer implements TemplateRenderer with custom ${var} syntax
type customTemplateRenderer struct {
	strictMode bool
}

// NewTemplateRenderer creates a new template renderer
func NewTemplateRenderer() TemplateRenderer {
	return &customTemplateRenderer{
		strictMode: true, // Default to strict mode
	}
}

// Render processes a template string and replaces ${variable} patterns
func (r *customTemplateRenderer) Render(ctx context.Context, template string, context map[string]interface{}) (string, error) {
	if context == nil {
		return "", ErrNilContext
	}

	result := strings.Builder{}
	i := 0
	templateLen := len(template)

	for i < templateLen {
		// Check for escape sequence
		if i < templateLen-1 && template[i] == '\\' && template[i+1] == '$' {
			result.WriteByte('$')
			i += 2
			continue
		}

		// Look for ${
		if i < templateLen-1 && template[i] == '$' && template[i+1] == '{' {
			// Find closing }
			end := strings.Index(template[i+2:], "}")
			if end == -1 {
				return "", fmt.Errorf("%w: unclosed brace in template", ErrInvalidTemplate)
			}
			end += i + 2 // Adjust to absolute position

			// Extract expression
			expr := template[i+2 : end]
			if expr == "" {
				return "", fmt.Errorf("%w: empty variable name", ErrInvalidTemplate)
			}

			// Evaluate expression
			value, err := r.evaluateExpression(expr, context)
			if err != nil {
				return "", err
			}

			// Convert value to string
			result.WriteString(fmt.Sprint(value))
			i = end + 1
			continue
		}

		// Regular character
		result.WriteByte(template[i])
		i++
	}

	return result.String(), nil
}

// evaluateExpression evaluates a template expression (variable access or function call)
func (r *customTemplateRenderer) evaluateExpression(expr string, context map[string]interface{}) (interface{}, error) {
	// Check for function call
	if strings.Contains(expr, "(") && strings.Contains(expr, ")") {
		return r.evaluateFunction(expr, context)
	}

	// Simple variable access
	return r.resolveVariable(expr, context)
}

// evaluateFunction evaluates a function call in the template
func (r *customTemplateRenderer) evaluateFunction(expr string, context map[string]interface{}) (interface{}, error) {
	// Parse function name and arguments
	openParen := strings.Index(expr, "(")
	if openParen == -1 {
		return nil, fmt.Errorf("%w: invalid function syntax", ErrInvalidTemplate)
	}

	funcName := strings.TrimSpace(expr[:openParen])
	argsStr := expr[openParen+1 : strings.LastIndex(expr, ")")]

	// Parse arguments
	args, err := r.parseArguments(argsStr, context)
	if err != nil {
		return nil, err
	}

	// Execute function
	return r.executeFunction(funcName, args, context)
}

// parseArguments parses function arguments
func (r *customTemplateRenderer) parseArguments(argsStr string, context map[string]interface{}) ([]interface{}, error) {
	if strings.TrimSpace(argsStr) == "" {
		return []interface{}{}, nil
	}

	// Simple comma split (doesn't handle nested commas in strings)
	parts := strings.Split(argsStr, ",")
	args := make([]interface{}, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)

		// Check if it's a string literal
		if strings.HasPrefix(part, "'") && strings.HasSuffix(part, "'") {
			args = append(args, part[1:len(part)-1])
			continue
		}

		// Check if it's a number
		if num, err := strconv.ParseFloat(part, 64); err == nil {
			if num == float64(int(num)) {
				args = append(args, int(num))
			} else {
				args = append(args, num)
			}
			continue
		}

		// Try to resolve as variable
		value, err := r.resolveVariable(part, context)
		if err != nil {
			if r.strictMode {
				return nil, err
			}
			args = append(args, nil)
			continue
		}
		args = append(args, value)
	}

	return args, nil
}

// executeFunction executes a built-in helper function
func (r *customTemplateRenderer) executeFunction(funcName string, args []interface{}, context map[string]interface{}) (interface{}, error) {
	switch funcName {
	case "upper":
		if len(args) != 1 {
			return nil, fmt.Errorf("upper requires 1 argument")
		}
		return strings.ToUpper(fmt.Sprint(args[0])), nil

	case "lower":
		if len(args) != 1 {
			return nil, fmt.Errorf("lower requires 1 argument")
		}
		return strings.ToLower(fmt.Sprint(args[0])), nil

	case "capitalize":
		if len(args) != 1 {
			return nil, fmt.Errorf("capitalize requires 1 argument")
		}
		s := fmt.Sprint(args[0])
		if len(s) == 0 {
			return "", nil
		}
		return strings.ToUpper(s[:1]) + s[1:], nil

	case "trim":
		if len(args) != 1 {
			return nil, fmt.Errorf("trim requires 1 argument")
		}
		return strings.TrimSpace(fmt.Sprint(args[0])), nil

	case "length":
		if len(args) != 1 {
			return nil, fmt.Errorf("length requires 1 argument")
		}
		val := reflect.ValueOf(args[0])
		switch val.Kind() {
		case reflect.Slice, reflect.Array, reflect.String, reflect.Map:
			return val.Len(), nil
		default:
			return nil, fmt.Errorf("%w: length() requires array, string, or map", ErrTypeMismatch)
		}

	case "default":
		if len(args) != 2 {
			return nil, fmt.Errorf("default requires 2 arguments")
		}
		if args[0] == nil || args[0] == "" {
			return args[1], nil
		}
		return args[0], nil

	case "join":
		if len(args) != 2 {
			return nil, fmt.Errorf("join requires 2 arguments")
		}
		val := reflect.ValueOf(args[0])
		if val.Kind() != reflect.Slice && val.Kind() != reflect.Array {
			return nil, fmt.Errorf("%w: join() requires array", ErrTypeMismatch)
		}

		sep := fmt.Sprint(args[1])
		parts := make([]string, val.Len())
		for i := 0; i < val.Len(); i++ {
			parts[i] = fmt.Sprint(val.Index(i).Interface())
		}
		return strings.Join(parts, sep), nil

	case "formatNumber":
		if len(args) != 2 {
			return nil, fmt.Errorf("formatNumber requires 2 arguments")
		}
		num, ok := args[0].(float64)
		if !ok {
			if intNum, ok := args[0].(int); ok {
				num = float64(intNum)
			} else {
				return nil, fmt.Errorf("%w: formatNumber() requires numeric value", ErrTypeMismatch)
			}
		}
		precision, ok := args[1].(int)
		if !ok {
			return nil, fmt.Errorf("%w: formatNumber() precision must be integer", ErrTypeMismatch)
		}
		return fmt.Sprintf("%."+strconv.Itoa(precision)+"f", num), nil

	case "formatDate":
		if len(args) != 2 {
			return nil, fmt.Errorf("formatDate requires 2 arguments")
		}
		dateStr := fmt.Sprint(args[0])
		layout := fmt.Sprint(args[1])

		// Try to parse the date
		t, err := time.Parse(time.RFC3339, dateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid date format: %v", err)
		}
		return t.Format(layout), nil

	case "if":
		if len(args) != 3 {
			return nil, fmt.Errorf("if requires 3 arguments")
		}
		condition, ok := args[0].(bool)
		if !ok {
			return nil, fmt.Errorf("%w: if() condition must be boolean", ErrTypeMismatch)
		}
		if condition {
			return args[1], nil
		}
		return args[2], nil

	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownFunction, funcName)
	}
}

// resolveVariable resolves a variable from the context using dot notation
func (r *customTemplateRenderer) resolveVariable(path string, context map[string]interface{}) (interface{}, error) {
	parts := strings.Split(path, ".")
	var current interface{} = context

	for i, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			val, ok := v[part]
			if !ok {
				if r.strictMode {
					return nil, fmt.Errorf("%w: %s", ErrUndefinedVariable, path)
				}
				return "", nil
			}
			current = val

		case map[interface{}]interface{}:
			val, ok := v[part]
			if !ok {
				if r.strictMode {
					return nil, fmt.Errorf("%w: %s", ErrUndefinedVariable, path)
				}
				return "", nil
			}
			current = val

		default:
			// Try reflection for struct fields
			rv := reflect.ValueOf(current)
			if rv.Kind() == reflect.Ptr {
				rv = rv.Elem()
			}
			if rv.Kind() == reflect.Struct {
				field := rv.FieldByName(part)
				if field.IsValid() {
					current = field.Interface()
					continue
				}
			}

			if r.strictMode && i < len(parts)-1 {
				return nil, fmt.Errorf("%w: %s", ErrUndefinedVariable, path)
			}
			return "", nil
		}
	}

	return current, nil
}
