package transform

import "errors"

// Sentinel errors shared across all transform operations
var (
	// JSONPath errors
	ErrInvalidJSONPath = errors.New("invalid JSONPath syntax")
	ErrTypeMismatch    = errors.New("type mismatch in JSONPath query")
	ErrNilData         = errors.New("cannot query nil data")

	// Expression errors
	ErrUnsafeOperation   = errors.New("unsafe operation attempted")
	ErrEvaluationTimeout = errors.New("expression evaluation timed out")
	ErrInvalidExpression = errors.New("invalid expression syntax")

	// Template and general errors
	ErrInvalidTemplate = errors.New("invalid template syntax")
	ErrUnknownFunction = errors.New("unknown template function")
	ErrNilContext      = errors.New("nil template context")
	ErrInvalidEscape   = errors.New("invalid escape sequence")

	// Shared undefined variable error
	ErrUndefinedVariable = errors.New("undefined variable")
)
