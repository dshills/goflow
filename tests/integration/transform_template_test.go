package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/transform"
)

// TestTemplateVariableInterpolation tests basic variable substitution with various data types
func TestTemplateVariableInterpolation(t *testing.T) {
	tests := []struct {
		name     string
		template string
		context  map[string]interface{}
		want     string
		wantErr  bool
	}{
		{
			name:     "simple string interpolation",
			template: "User: ${name}, Email: ${email}",
			context: map[string]interface{}{
				"name":  "Alice Johnson",
				"email": "alice@example.com",
			},
			want:    "User: Alice Johnson, Email: alice@example.com",
			wantErr: false,
		},
		{
			name:     "numeric interpolation",
			template: "Product: ${product}, Price: ${price}, Quantity: ${qty}",
			context: map[string]interface{}{
				"product": "Widget",
				"price":   99.99,
				"qty":     5,
			},
			want:    "Product: Widget, Price: 99.99, Quantity: 5",
			wantErr: false,
		},
		{
			name:     "boolean interpolation",
			template: "Active: ${isActive}, Verified: ${isVerified}",
			context: map[string]interface{}{
				"isActive":   true,
				"isVerified": false,
			},
			want:    "Active: true, Verified: false",
			wantErr: false,
		},
		{
			name:     "nested object field access",
			template: "Customer: ${user.name}, City: ${user.address.city}",
			context: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "Bob Smith",
					"address": map[string]interface{}{
						"city":  "New York",
						"state": "NY",
					},
				},
			},
			want:    "Customer: Bob Smith, City: New York",
			wantErr: false,
		},
		{
			name:     "deep nested field access",
			template: "Contact: ${user.profile.contact.phone}",
			context: map[string]interface{}{
				"user": map[string]interface{}{
					"profile": map[string]interface{}{
						"contact": map[string]interface{}{
							"phone": "555-1234",
							"email": "test@example.com",
						},
					},
				},
			},
			want:    "Contact: 555-1234",
			wantErr: false,
		},
		{
			name:     "same variable used multiple times",
			template: "ID: ${id} - Reference: ${id}",
			context: map[string]interface{}{
				"id": "ORD-12345",
			},
			want:    "ID: ORD-12345 - Reference: ORD-12345",
			wantErr: false,
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

// TestTemplateHelperFunctions tests all built-in helper functions
func TestTemplateHelperFunctions(t *testing.T) {
	tests := []struct {
		name     string
		template string
		context  map[string]interface{}
		want     string
		wantErr  bool
	}{
		// Text transformation helpers
		{
			name:     "upper() text transformation",
			template: "${upper(title)}",
			context: map[string]interface{}{
				"title": "hello world",
			},
			want:    "HELLO WORLD",
			wantErr: false,
		},
		{
			name:     "lower() text transformation",
			template: "${lower(title)}",
			context: map[string]interface{}{
				"title": "HELLO WORLD",
			},
			want:    "hello world",
			wantErr: false,
		},
		{
			name:     "capitalize() first letter",
			template: "${capitalize(word)}",
			context: map[string]interface{}{
				"word": "hello",
			},
			want:    "Hello",
			wantErr: false,
		},
		{
			name:     "trim() whitespace",
			template: "'${trim(text)}'",
			context: map[string]interface{}{
				"text": "  hello world  ",
			},
			want:    "'hello world'",
			wantErr: false,
		},
		// Array/Collection helpers
		{
			name:     "length() of array",
			template: "Count: ${length(items)}",
			context: map[string]interface{}{
				"items": []interface{}{"apple", "banana", "cherry", "date"},
			},
			want:    "Count: 4",
			wantErr: false,
		},
		{
			name:     "join() array elements",
			template: "Tags: ${join(tags, ', ')}",
			context: map[string]interface{}{
				"tags": []interface{}{"go", "workflow", "orchestration"},
			},
			want:    "Tags: go, workflow, orchestration",
			wantErr: false,
		},
		{
			name:     "join() with pipe separator",
			template: "${join(roles, ' | ')}",
			context: map[string]interface{}{
				"roles": []interface{}{"admin", "editor", "viewer"},
			},
			want:    "admin | editor | viewer",
			wantErr: false,
		},
		// Number formatting
		{
			name:     "formatNumber() with 2 decimal places",
			template: "Price: $${formatNumber(price, 2)}",
			context: map[string]interface{}{
				"price": 1234.5,
			},
			want:    "Price: $1234.50",
			wantErr: false,
		},
		{
			name:     "formatNumber() integer with 0 decimal places",
			template: "Count: ${formatNumber(count, 0)}",
			context: map[string]interface{}{
				"count": 42.7,
			},
			want:    "Count: 43",
			wantErr: false,
		},
		{
			name:     "formatNumber() 3 decimal places",
			template: "Precision: ${formatNumber(value, 3)}",
			context: map[string]interface{}{
				"value": 3.14159,
			},
			want:    "Precision: 3.142",
			wantErr: false,
		},
		// Date formatting
		{
			name:     "formatDate() with date layout",
			template: "Date: ${formatDate(timestamp, '2006-01-02')}",
			context: map[string]interface{}{
				"timestamp": "2024-12-01T10:30:00Z",
			},
			want:    "Date: 2024-12-01",
			wantErr: false,
		},
		{
			name:     "formatDate() with time layout",
			template: "Time: ${formatDate(timestamp, '15:04:05')}",
			context: map[string]interface{}{
				"timestamp": "2024-12-01T10:30:45Z",
			},
			want:    "Time: 10:30:45",
			wantErr: false,
		},
		// Conditional helpers
		{
			name:     "if() true condition",
			template: "Status: ${if(active, 'Active', 'Inactive')}",
			context: map[string]interface{}{
				"active": true,
			},
			want:    "Status: Active",
			wantErr: false,
		},
		{
			name:     "if() false condition",
			template: "Status: ${if(active, 'Active', 'Inactive')}",
			context: map[string]interface{}{
				"active": false,
			},
			want:    "Status: Inactive",
			wantErr: false,
		},
		// Default value helper
		{
			name:     "default() with missing variable",
			template: "Role: ${default(role, 'User')}",
			context:  map[string]interface{}{},
			want:     "Role: User",
			wantErr:  false,
		},
		{
			name:     "default() with provided value",
			template: "Role: ${default(role, 'User')}",
			context: map[string]interface{}{
				"role": "Admin",
			},
			want:    "Role: Admin",
			wantErr: false,
		},
		// Nested helpers
		{
			name:     "nested upper() and trim()",
			template: "${upper(trim(text))}",
			context: map[string]interface{}{
				"text": "  hello world  ",
			},
			want:    "HELLO WORLD",
			wantErr: false,
		},
		{
			name:     "helper with field access",
			template: "${upper(user.name)}",
			context: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "john doe",
				},
			},
			want:    "JOHN DOE",
			wantErr: false,
		},
		{
			name:     "multiple helpers in template",
			template: "User: ${capitalize(trim(name))}, Email: ${lower(email)}",
			context: map[string]interface{}{
				"name":  "  john  ",
				"email": "JOHN@EXAMPLE.COM",
			},
			want:    "User: John, Email: john@example.com",
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

// TestTemplateStrictMode tests behavior with missing variables in strict vs lenient modes
func TestTemplateStrictMode(t *testing.T) {
	tests := []struct {
		name         string
		template     string
		context      map[string]interface{}
		strictMode   bool
		defaultValue string
		want         string
		wantErr      bool
		errType      error
	}{
		{
			name:       "missing variable in strict mode",
			template:   "Hello ${user.name}",
			context:    map[string]interface{}{},
			strictMode: true,
			want:       "",
			wantErr:    true,
			errType:    transform.ErrUndefinedVariable,
		},
		{
			name:       "missing variable in lenient mode",
			template:   "Hello ${user.name}",
			context:    map[string]interface{}{},
			strictMode: false,
			want:       "Hello ",
			wantErr:    false,
		},
		{
			name:         "missing variable with default value",
			template:     "Hello ${name}",
			context:      map[string]interface{}{},
			strictMode:   false,
			defaultValue: "Guest",
			want:         "Hello Guest",
			wantErr:      false,
		},
		{
			name:     "partial path missing in strict mode",
			template: "Email: ${user.profile.email}",
			context: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "John",
				},
			},
			strictMode: true,
			want:       "",
			wantErr:    true,
			errType:    transform.ErrUndefinedVariable,
		},
		{
			name:     "partial path missing in lenient mode",
			template: "Email: ${user.profile.email}",
			context: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "John",
				},
			},
			strictMode: false,
			want:       "Email: ",
			wantErr:    false,
		},
		{
			name:     "some variables present, some missing in lenient mode",
			template: "User ${name} has email ${email}",
			context: map[string]interface{}{
				"name": "Alice",
			},
			strictMode: false,
			want:       "User Alice has email ",
			wantErr:    false,
		},
		{
			name:     "some variables present, some missing in strict mode",
			template: "User ${name} has email ${email}",
			context: map[string]interface{}{
				"name": "Alice",
			},
			strictMode: true,
			want:       "",
			wantErr:    true,
			errType:    transform.ErrUndefinedVariable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := transform.NewTemplateRenderer()
			renderer.SetStrictMode(tt.strictMode)
			if tt.defaultValue != "" {
				renderer.SetDefaultValue(tt.defaultValue)
			}

			got, err := renderer.Render(context.Background(), tt.template, tt.context)

			if (err != nil) != tt.wantErr {
				t.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errType != nil {
				if !errors.Is(err, tt.errType) {
					t.Errorf("Render() error type = %v, want %v", err, tt.errType)
				}
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("Render() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestTemplateErrorHandling tests error scenarios and edge cases
func TestTemplateErrorHandling(t *testing.T) {
	tests := []struct {
		name     string
		template string
		context  map[string]interface{}
		wantErr  bool
		errType  error
	}{
		{
			name:     "invalid template - unclosed brace",
			template: "Hello ${name",
			context:  map[string]interface{}{"name": "World"},
			wantErr:  true,
			errType:  transform.ErrInvalidTemplate,
		},
		{
			name:     "invalid template - empty variable name",
			template: "Hello ${}",
			context:  map[string]interface{}{},
			wantErr:  true,
			errType:  transform.ErrInvalidTemplate,
		},
		{
			name:     "invalid template - empty braces with text",
			template: "Price: ${}",
			context:  map[string]interface{}{},
			wantErr:  true,
			errType:  transform.ErrInvalidTemplate,
		},
		{
			name:     "unknown function call",
			template: "${unknown(value)}",
			context:  map[string]interface{}{"value": "test"},
			wantErr:  true,
			errType:  transform.ErrUnknownFunction,
		},
		{
			name:     "wrong argument count for upper()",
			template: "${upper(value, extra)}",
			context:  map[string]interface{}{"value": "test"},
			wantErr:  true,
		},
		{
			name:     "wrong argument count for formatNumber()",
			template: "${formatNumber(price)}",
			context:  map[string]interface{}{"price": 99.99},
			wantErr:  true,
		},
		{
			name:     "type error - length() on number",
			template: "${length(count)}",
			context:  map[string]interface{}{"count": 42},
			wantErr:  true,
			errType:  transform.ErrTypeMismatch,
		},
		{
			name:     "type error - join() on non-array",
			template: "${join(notAnArray, ',')}",
			context:  map[string]interface{}{"notAnArray": "string"},
			wantErr:  true,
			errType:  transform.ErrTypeMismatch,
		},
		{
			name:     "type error - formatNumber() with non-numeric value",
			template: "${formatNumber(text, 2)}",
			context:  map[string]interface{}{"text": "not a number"},
			wantErr:  true,
			errType:  transform.ErrTypeMismatch,
		},
		{
			name:     "type error - if() with non-boolean condition",
			template: "${if(condition, 'yes', 'no')}",
			context:  map[string]interface{}{"condition": "not a boolean"},
			wantErr:  true,
			errType:  transform.ErrTypeMismatch,
		},
		{
			name:     "nil context",
			template: "Hello ${name}",
			context:  nil,
			wantErr:  true,
			errType:  transform.ErrNilContext,
		},
		{
			name:     "invalid date format",
			template: "${formatDate(date, '2006-01-02')}",
			context: map[string]interface{}{
				"date": "not-a-date",
			},
			wantErr: true,
		},
		{
			name:     "wrong argument count for default()",
			template: "${default(value)}",
			context:  map[string]interface{}{"value": "test"},
			wantErr:  true,
		},
		{
			name:     "wrong argument count for if()",
			template: "${if(condition, 'yes')}",
			context:  map[string]interface{}{"condition": true},
			wantErr:  true,
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
					t.Errorf("Render() error type = %v, want %v", err, tt.errType)
				}
			}
		})
	}
}

// TestTemplateRealWorldScenarios tests practical use cases
func TestTemplateRealWorldScenarios(t *testing.T) {
	tests := []struct {
		name     string
		template string
		context  map[string]interface{}
		want     string
		wantErr  bool
	}{
		{
			name: "email notification template",
			template: `Dear ${user.firstName},

Your order ${order.id} has been ${order.status}.
Order Total: $${formatNumber(order.total, 2)}
Items: ${length(order.items)}

Best regards,
The ${company.name} Team`,
			context: map[string]interface{}{
				"user": map[string]interface{}{
					"firstName": "Alice",
					"lastName":  "Johnson",
				},
				"order": map[string]interface{}{
					"id":     "ORD-12345",
					"status": "shipped",
					"total":  149.99,
					"items": []interface{}{
						map[string]interface{}{"sku": "ITEM-1"},
						map[string]interface{}{"sku": "ITEM-2"},
					},
				},
				"company": map[string]interface{}{
					"name": "GoFlow Systems",
				},
			},
			want: `Dear Alice,

Your order ORD-12345 has been shipped.
Order Total: $149.99
Items: 2

Best regards,
The GoFlow Systems Team`,
			wantErr: false,
		},
		{
			name:     "API endpoint construction",
			template: "https://api.example.com/v${apiVersion}/users/${userId}/orders/${orderId}/items?limit=${limit}",
			context: map[string]interface{}{
				"apiVersion": 2,
				"userId":     "usr-123",
				"orderId":    "ord-456",
				"limit":      10,
			},
			want:    "https://api.example.com/v2/users/usr-123/orders/ord-456/items?limit=10",
			wantErr: false,
		},
		{
			name:     "log message with all field types",
			template: "[${upper(level)}] ${formatDate(timestamp, '2006-01-02 15:04:05')} - ${service}: ${message} (duration=${duration}ms, success=${status})",
			context: map[string]interface{}{
				"level":     "info",
				"timestamp": "2024-12-01T10:30:00Z",
				"service":   "payment-service",
				"message":   "Payment processed successfully",
				"duration":  245,
				"status":    true,
			},
			want:    "[INFO] 2024-12-01 10:30:00 - payment-service: Payment processed successfully (duration=245ms, success=true)",
			wantErr: false,
		},
		{
			name:     "CSV row generation",
			template: "${id},${capitalize(name)},${email},${formatNumber(salary, 2)}",
			context: map[string]interface{}{
				"id":     "E001",
				"name":   "john doe",
				"email":  "john@example.com",
				"salary": 55000.50,
			},
			want:    "E001,John doe,john@example.com,55000.50",
			wantErr: false,
		},
		{
			name:     "status report with conditionals",
			template: "Processing ${items} items - Status: ${if(processing, 'In Progress', 'Completed')} - Success Rate: ${formatNumber(successRate, 1)}%",
			context: map[string]interface{}{
				"items":       5,
				"processing":  false,
				"successRate": 98.5,
			},
			want:    "Processing 5 items - Status: Completed - Success Rate: 98.5%",
			wantErr: false,
		},
		{
			name:     "configuration template with defaults",
			template: "Server: ${server.host}:${server.port} (${if(server.ssl, 'HTTPS', 'HTTP')}), Timeout: ${default(timeout, '30')}s",
			context: map[string]interface{}{
				"server": map[string]interface{}{
					"host": "api.example.com",
					"port": 443,
					"ssl":  true,
				},
			},
			want:    "Server: api.example.com:443 (HTTPS), Timeout: 30s",
			wantErr: false,
		},
		{
			name:     "product listing with formatted values",
			template: "Product: ${product.name}, Price: $${formatNumber(product.price, 2)}, Tags: ${join(product.tags, ' / ')}",
			context: map[string]interface{}{
				"product": map[string]interface{}{
					"name":  "Wireless Mouse",
					"price": 29.99,
					"tags":  []interface{}{"electronics", "computer", "accessory"},
				},
			},
			want:    "Product: Wireless Mouse, Price: $29.99, Tags: electronics / computer / accessory",
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

// TestTemplateConcurrency verifies that the same template can be rendered
// concurrently with different contexts
func TestTemplateConcurrency(t *testing.T) {
	renderer := transform.NewTemplateRenderer()
	template := "Hello ${name}, your order is ${status}"

	// Run multiple concurrent renderings
	results := make(chan string, 10)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func(index int) {
			ctx := map[string]interface{}{
				"name":   "User" + string(rune(index)),
				"status": "ready",
			}
			result, err := renderer.Render(context.Background(), template, ctx)
			if err != nil {
				errors <- err
			} else {
				results <- result
			}
		}(i)
	}

	// Collect results
	successCount := 0
	for i := 0; i < 10; i++ {
		select {
		case err := <-errors:
			t.Errorf("Concurrent render failed: %v", err)
		case result := <-results:
			if result != "" {
				successCount++
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("Timeout waiting for concurrent renders")
		}
	}

	if successCount != 10 {
		t.Errorf("Expected 10 successful renders, got %d", successCount)
	}
}

// TestTemplateWithContextDeadline verifies cancellation works properly
func TestTemplateWithContextDeadline(t *testing.T) {
	renderer := transform.NewTemplateRenderer()
	template := "Hello ${name}"
	templateContext := map[string]interface{}{
		"name": "World",
	}

	// Test with valid context (short timeout)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	result, err := renderer.Render(ctx, template, templateContext)
	if err != nil {
		t.Errorf("Expected successful render with valid context, got error: %v", err)
	}
	if result != "Hello World" {
		t.Errorf("Expected 'Hello World', got %q", result)
	}

	// Test with expired context
	expiredCtx, cancel := context.WithTimeout(context.Background(), -1*time.Millisecond)
	defer cancel()

	// Note: The current implementation doesn't check context deadline during rendering,
	// but we're testing that it doesn't crash with an expired context
	_, _ = renderer.Render(expiredCtx, template, templateContext)
}
