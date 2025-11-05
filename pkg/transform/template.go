package transform

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// TemplateRenderer defines the interface for rendering templates
//
// Thread-safety: Implementations are NOT goroutine-safe. Configuration methods
// (SetStrictMode, SetDefaultValue) must not be called concurrently with Render.
// Configure the renderer before use, then use it from multiple goroutines, or
// create separate renderer instances per goroutine.
type TemplateRenderer interface {
	Render(ctx context.Context, template string, context map[string]interface{}) (string, error)
	SetStrictMode(strict bool)
	SetDefaultValue(defaultVal string)
}

// customTemplateRenderer implements TemplateRenderer with custom ${var} syntax
type customTemplateRenderer struct {
	strictMode   bool
	defaultValue string
}

// NewTemplateRenderer creates a new template renderer
// By default, uses lenient mode (strictMode=false) to allow missing variables
func NewTemplateRenderer() TemplateRenderer {
	return &customTemplateRenderer{
		strictMode:   false, // Default to lenient mode for flexibility
		defaultValue: "",    // Default to empty string for missing vars
	}
}

// SetStrictMode configures whether to fail on missing variables
func (r *customTemplateRenderer) SetStrictMode(strict bool) {
	r.strictMode = strict
}

// SetDefaultValue sets the default value for missing variables in non-strict mode
func (r *customTemplateRenderer) SetDefaultValue(defaultVal string) {
	r.defaultValue = defaultVal
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

	// Smart split that respects quoted strings
	parts := smartSplitArgs(argsStr)
	args := make([]interface{}, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)

		// Check if it's a string literal (single or double quotes)
		if len(part) >= 2 {
			if (strings.HasPrefix(part, "'") && strings.HasSuffix(part, "'")) ||
				(strings.HasPrefix(part, "\"") && strings.HasSuffix(part, "\"")) {
				args = append(args, part[1:len(part)-1])
				continue
			}
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

		// Check if it's a boolean
		if part == "true" {
			args = append(args, true)
			continue
		}
		if part == "false" {
			args = append(args, false)
			continue
		}

		// Check if it's a nested function call
		if strings.Contains(part, "(") && strings.Contains(part, ")") {
			value, err := r.evaluateFunction(part, context)
			if err != nil {
				// If function evaluation fails, try as variable
				value, err = r.resolveVariable(part, context)
				if err != nil {
					args = append(args, nil)
					continue
				}
			}
			args = append(args, value)
			continue
		}

		// Try to resolve as variable
		// For default() function, we allow first arg to be missing
		value, err := r.resolveVariable(part, context)
		if err != nil {
			// Don't fail immediately - let the function handle nil args
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
		if len(args) < 2 {
			return nil, fmt.Errorf("default requires 2 arguments")
		}
		// Return the default value (second arg) if first arg is nil or empty
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
				if r.defaultValue != "" {
					return r.defaultValue, nil
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
				if r.defaultValue != "" {
					return r.defaultValue, nil
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
			if r.defaultValue != "" {
				return r.defaultValue, nil
			}
			return "", nil
		}
	}

	return current, nil
}

// smartSplitArgs splits function arguments by comma, but respects quoted strings
func smartSplitArgs(argsStr string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for i, ch := range argsStr {
		if (ch == '\'' || ch == '"') && (i == 0 || argsStr[i-1] != '\\') {
			if !inQuote {
				inQuote = true
				quoteChar = ch
			} else if ch == quoteChar {
				inQuote = false
				quoteChar = 0
			}
			current.WriteRune(ch)
		} else if ch == ',' && !inQuote {
			parts = append(parts, current.String())
			current.Reset()
		} else {
			current.WriteRune(ch)
		}
	}

	// Add the last part
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}
