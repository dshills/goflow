package transform_test

import (
	"context"
	"errors"

	"github.com/dshills/goflow/pkg/transform"

	"testing"
)

// TestBasicTemplateInterpolation tests simple variable substitution
func TestBasicTemplateInterpolation(t *testing.T) {
	tests := []struct {
		name     string
		template string
		context  map[string]interface{}
		want     string
		wantErr  bool
	}{
		{
			name:     "single variable substitution",
			template: "Hello ${user.name}",
			context: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "John",
				},
			},
			want:    "Hello John",
			wantErr: false,
		},
		{
			name:     "multiple variable substitution",
			template: "Hello ${user.name}, order ${order.id} is ready",
			context: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "Alice",
				},
				"order": map[string]interface{}{
					"id": "12345",
				},
			},
			want:    "Hello Alice, order 12345 is ready",
			wantErr: false,
		},
		{
			name:     "nested field access",
			template: "Email: ${user.profile.contact.email}",
			context: map[string]interface{}{
				"user": map[string]interface{}{
					"profile": map[string]interface{}{
						"contact": map[string]interface{}{
							"email": "alice@example.com",
						},
					},
				},
			},
			want:    "Email: alice@example.com",
			wantErr: false,
		},
		{
			name:     "numeric variable",
			template: "Total: ${price} USD",
			context: map[string]interface{}{
				"price": 99.99,
			},
			want:    "Total: 99.99 USD",
			wantErr: false,
		},
		{
			name:     "boolean variable",
			template: "Active: ${status.active}",
			context: map[string]interface{}{
				"status": map[string]interface{}{
					"active": true,
				},
			},
			want:    "Active: true",
			wantErr: false,
		},
		{
			name:     "template with no variables",
			template: "This is a plain string",
			context:  map[string]interface{}{},
			want:     "This is a plain string",
			wantErr:  false,
		},
		{
			name:     "adjacent variables",
			template: "${firstName}${lastName}",
			context: map[string]interface{}{
				"firstName": "John",
				"lastName":  "Doe",
			},
			want:    "JohnDoe",
			wantErr: false,
		},
		{
			name:     "variables with surrounding text",
			template: "The ${animal} jumped over the ${obstacle}.",
			context: map[string]interface{}{
				"animal":   "fox",
				"obstacle": "fence",
			},
			want:    "The fox jumped over the fence.",
			wantErr: false,
		},
		{
			name:     "same variable used multiple times",
			template: "${name} said: 'Hi, I'm ${name}'",
			context: map[string]interface{}{
				"name": "Bob",
			},
			want:    "Bob said: 'Hi, I'm Bob'",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := transform.NewTemplateRenderer()
			got, err := renderer.Render(context.Background(), tt.template, tt.context)

			if (err != nil) != tt.wantErr {
				t.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("Render() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestMissingVariableHandling tests behavior when variables are not found in context
func TestMissingVariableHandling(t *testing.T) {
	tests := []struct {
		name         string
		template     string
		context      map[string]interface{}
		want         string
		wantErr      bool
		errType      error
		strictMode   bool   // Whether to fail on missing variables
		defaultValue string // Default value for missing variables
	}{
		{
			name:       "missing variable in strict mode",
			template:   "Hello ${user.name}",
			context:    map[string]interface{}{},
			want:       "",
			wantErr:    true,
			errType:    transform.ErrUndefinedVariable,
			strictMode: true,
		},
		{
			name:         "missing variable with default value",
			template:     "Hello ${user.name}",
			context:      map[string]interface{}{},
			want:         "Hello <undefined>",
			wantErr:      false,
			strictMode:   false,
			defaultValue: "<undefined>",
		},
		{
			name:     "partially missing nested field",
			template: "Email: ${user.profile.email}",
			context: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "John",
				},
			},
			want:       "",
			wantErr:    true,
			errType:    transform.ErrUndefinedVariable,
			strictMode: true,
		},
		{
			name:     "missing variable with lenient mode",
			template: "Status: ${status}",
			context: map[string]interface{}{
				"other": "value",
			},
			want:       "Status: ",
			wantErr:    false,
			strictMode: false,
		},
		{
			name:     "some variables present, some missing in lenient mode",
			template: "User ${name} has email ${email}",
			context: map[string]interface{}{
				"name": "Alice",
			},
			want:       "User Alice has email ",
			wantErr:    false,
			strictMode: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := transform.NewTemplateRenderer()
			// Configure renderer based on test case
			renderer.SetStrictMode(tt.strictMode)
			renderer.SetDefaultValue(tt.defaultValue)

			got, err := renderer.Render(context.Background(), tt.template, tt.context)

			if (err != nil) != tt.wantErr {
				t.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errType != nil {
				if !errors.Is(err, tt.errType) {
					t.Errorf("Render() error = %v, want error type %v", err, tt.errType)
				}
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("Render() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestTemplateHelperFunctions tests built-in helper functions
func TestTemplateHelperFunctions(t *testing.T) {
	tests := []struct {
		name     string
		template string
		context  map[string]interface{}
		want     string
		wantErr  bool
	}{
		{
			name:     "uppercase helper",
			template: "${upper(name)}",
			context: map[string]interface{}{
				"name": "john",
			},
			want:    "JOHN",
			wantErr: false,
		},
		{
			name:     "lowercase helper",
			template: "${lower(name)}",
			context: map[string]interface{}{
				"name": "ALICE",
			},
			want:    "alice",
			wantErr: false,
		},
		{
			name:     "capitalize helper",
			template: "${capitalize(word)}",
			context: map[string]interface{}{
				"word": "hello",
			},
			want:    "Hello",
			wantErr: false,
		},
		{
			name:     "trim helper",
			template: "${trim(text)}",
			context: map[string]interface{}{
				"text": "  hello world  ",
			},
			want:    "hello world",
			wantErr: false,
		},
		{
			name:     "length helper",
			template: "Count: ${length(items)}",
			context: map[string]interface{}{
				"items": []interface{}{"a", "b", "c"},
			},
			want:    "Count: 3",
			wantErr: false,
		},
		{
			name:     "default value helper",
			template: "${default(missing, 'N/A')}",
			context:  map[string]interface{}{},
			want:     "N/A",
			wantErr:  false,
		},
		{
			name:     "join array helper",
			template: "${join(items, ', ')}",
			context: map[string]interface{}{
				"items": []interface{}{"apple", "banana", "cherry"},
			},
			want:    "apple, banana, cherry",
			wantErr: false,
		},
		{
			name:     "format number helper",
			template: "${formatNumber(price, 2)}",
			context: map[string]interface{}{
				"price": 1234.5,
			},
			want:    "1234.50",
			wantErr: false,
		},
		{
			name:     "date format helper",
			template: "${formatDate(timestamp, '2006-01-02')}",
			context: map[string]interface{}{
				"timestamp": "2024-12-01T10:30:00Z",
			},
			want:    "2024-12-01",
			wantErr: false,
		},
		{
			name:     "conditional helper",
			template: "${if(active, 'Active', 'Inactive')}",
			context: map[string]interface{}{
				"active": true,
			},
			want:    "Active",
			wantErr: false,
		},
		{
			name:     "nested helpers",
			template: "${upper(trim(text))}",
			context: map[string]interface{}{
				"text": "  hello  ",
			},
			want:    "HELLO",
			wantErr: false,
		},
		{
			name:     "helper with field access",
			template: "${upper(user.name)}",
			context: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "john",
				},
			},
			want:    "JOHN",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := transform.NewTemplateRenderer()
			got, err := renderer.Render(context.Background(), tt.template, tt.context)

			if (err != nil) != tt.wantErr {
				t.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("Render() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestTemplateComplexScenarios tests real-world template usage
func TestTemplateComplexScenarios(t *testing.T) {
	tests := []struct {
		name     string
		template string
		context  map[string]interface{}
		want     string
		wantErr  bool
	}{
		{
			name: "email notification template",
			template: `Hello ${user.name},

Your order ${order.id} has been ${order.status}.
Total: $${order.total}

Thank you for your purchase!`,
			context: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "Alice Johnson",
				},
				"order": map[string]interface{}{
					"id":     "ORD-12345",
					"status": "shipped",
					"total":  149.99,
				},
			},
			want: `Hello Alice Johnson,

Your order ORD-12345 has been shipped.
Total: $149.99

Thank you for your purchase!`,
			wantErr: false,
		},
		{
			name:     "API endpoint construction",
			template: "https://api.example.com/v${apiVersion}/users/${userId}/orders/${orderId}",
			context: map[string]interface{}{
				"apiVersion": 2,
				"userId":     "user-123",
				"orderId":    "ord-456",
			},
			want:    "https://api.example.com/v2/users/user-123/orders/ord-456",
			wantErr: false,
		},
		{
			name:     "log message with multiple fields",
			template: "[${level}] ${timestamp} - ${service}: ${message} (user=${user.id}, duration=${duration}ms)",
			context: map[string]interface{}{
				"level":     "INFO",
				"timestamp": "2024-12-01T10:30:00Z",
				"service":   "payment-service",
				"message":   "Payment processed successfully",
				"user": map[string]interface{}{
					"id": "usr-789",
				},
				"duration": 245,
			},
			want:    "[INFO] 2024-12-01T10:30:00Z - payment-service: Payment processed successfully (user=usr-789, duration=245ms)",
			wantErr: false,
		},
		{
			name:     "SQL query construction (parameterized)",
			template: "SELECT * FROM ${table} WHERE ${field} = '${value}' LIMIT ${limit}",
			context: map[string]interface{}{
				"table": "users",
				"field": "email",
				"value": "alice@example.com",
				"limit": 10,
			},
			want:    "SELECT * FROM users WHERE email = 'alice@example.com' LIMIT 10",
			wantErr: false,
		},
		{
			name:     "markdown document generation",
			template: "# ${title}\n\n## Author: ${author}\n\nPublished: ${publishDate}\n\n${content}",
			context: map[string]interface{}{
				"title":       "GoFlow Documentation",
				"author":      "John Doe",
				"publishDate": "2024-12-01",
				"content":     "This is the content of the document.",
			},
			want:    "# GoFlow Documentation\n\n## Author: John Doe\n\nPublished: 2024-12-01\n\nThis is the content of the document.",
			wantErr: false,
		},
		{
			name:     "JSON-like structure construction",
			template: `{"name":"${name}","email":"${email}","role":"${role}"}`,
			context: map[string]interface{}{
				"name":  "Bob Smith",
				"email": "bob@example.com",
				"role":  "admin",
			},
			want:    `{"name":"Bob Smith","email":"bob@example.com","role":"admin"}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := transform.NewTemplateRenderer()
			got, err := renderer.Render(context.Background(), tt.template, tt.context)

			if (err != nil) != tt.wantErr {
				t.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("Render() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestTemplateSyntaxVariations tests different template syntax styles
func TestTemplateSyntaxVariations(t *testing.T) {
	tests := []struct {
		name     string
		template string
		context  map[string]interface{}
		want     string
		wantErr  bool
	}{
		{
			name:     "dollar brace syntax",
			template: "Hello ${name}",
			context: map[string]interface{}{
				"name": "World",
			},
			want:    "Hello World",
			wantErr: false,
		},
		{
			name:     "escaped dollar sign",
			template: "Price: \\${price}",
			context: map[string]interface{}{
				"price": 99,
			},
			want:    "Price: ${price}",
			wantErr: false,
		},
		{
			name:     "literal dollar without brace",
			template: "Cost is $100",
			context:  map[string]interface{}{},
			want:     "Cost is $100",
			wantErr:  false,
		},
		{
			name:     "empty braces",
			template: "Hello ${}",
			context:  map[string]interface{}{},
			want:     "",
			wantErr:  true,
		},
		{
			name:     "unclosed brace",
			template: "Hello ${name",
			context: map[string]interface{}{
				"name": "World",
			},
			want:    "",
			wantErr: true,
		},
		{
			name:     "multiple dollars",
			template: "$$${price}",
			context: map[string]interface{}{
				"price": 50,
			},
			want:    "$$50",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := transform.NewTemplateRenderer()
			got, err := renderer.Render(context.Background(), tt.template, tt.context)

			if (err != nil) != tt.wantErr {
				t.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("Render() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestTemplateErrorHandling tests error scenarios
func TestTemplateErrorHandling(t *testing.T) {
	tests := []struct {
		name     string
		template string
		context  map[string]interface{}
		wantErr  bool
		errType  error
	}{
		{
			name:     "invalid syntax - unclosed brace",
			template: "${name",
			context:  map[string]interface{}{"name": "test"},
			wantErr:  true,
			errType:  transform.ErrInvalidTemplate,
		},
		{
			name:     "invalid syntax - empty variable name",
			template: "${}",
			context:  map[string]interface{}{},
			wantErr:  true,
			errType:  transform.ErrInvalidTemplate,
		},
		{
			name:     "invalid function call",
			template: "${unknown(value)}",
			context:  map[string]interface{}{"value": "test"},
			wantErr:  true,
			errType:  transform.ErrUnknownFunction,
		},
		{
			name:     "nil context",
			template: "Hello ${name}",
			context:  nil,
			wantErr:  true,
			errType:  transform.ErrNilContext,
		},
		{
			name:     "type conversion error in helper",
			template: "${length(notAnArray)}",
			context:  map[string]interface{}{"notAnArray": 42},
			wantErr:  true,
			errType:  transform.ErrTypeMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := transform.NewTemplateRenderer()
			_, err := renderer.Render(context.Background(), tt.template, tt.context)

			if (err != nil) != tt.wantErr {
				t.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errType != nil {
				if !errors.Is(err, tt.errType) {
					t.Errorf("Render() error = %v, want error type %v", err, tt.errType)
				}
			}
		})
	}
}

// TestTemplatePerformance tests performance characteristics
func TestTemplatePerformance(t *testing.T) {
	// Test that templates can be compiled/cached for reuse
	t.Run("template caching", func(t *testing.T) {
		renderer := transform.NewTemplateRenderer()
		template := "Hello ${user.name}, your balance is ${account.balance}"

		// First render
		ctx1 := map[string]interface{}{
			"user":    map[string]interface{}{"name": "Alice"},
			"account": map[string]interface{}{"balance": 1000.50},
		}
		_, err := renderer.Render(context.Background(), template, ctx1)
		if err != nil {
			t.Errorf("First render failed: %v", err)
		}

		// Second render with different context (should use cached template)
		ctx2 := map[string]interface{}{
			"user":    map[string]interface{}{"name": "Bob"},
			"account": map[string]interface{}{"balance": 2500.75},
		}
		_, err = renderer.Render(context.Background(), template, ctx2)
		if err != nil {
			t.Errorf("Second render failed: %v", err)
		}

		// Real performance test would measure time difference
	})

	t.Run("large template", func(t *testing.T) {
		renderer := transform.NewTemplateRenderer()

		// Build a large template with many substitutions
		template := ""
		templateContext := make(map[string]interface{})
		for i := 0; i < 100; i++ {
			template += "Value " + string(rune(i)) + ": ${var" + string(rune(i)) + "}\n"
			templateContext["var"+string(rune(i))] = i
		}

		_, err := renderer.Render(context.Background(), template, templateContext)
		if err != nil {
			t.Errorf("Large template render failed: %v", err)
		}
	})
}
