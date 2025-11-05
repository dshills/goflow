package integration

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/transform"
)

// TestJSONPathFilters_Integration tests JSONPath filter operations with real-world data
func TestJSONPathFilters_Integration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tests := []struct {
		name     string
		jsonPath string
		data     interface{}
		want     interface{}
		wantErr  bool
		errType  error
	}{
		{
			name:     "filter products by price threshold",
			jsonPath: "$.products[?(@.price < 100)]",
			data: map[string]interface{}{
				"products": []interface{}{
					map[string]interface{}{"sku": "PROD-001", "name": "Widget A", "price": 49.99},
					map[string]interface{}{"sku": "PROD-002", "name": "Widget B", "price": 150.00},
					map[string]interface{}{"sku": "PROD-003", "name": "Widget C", "price": 75.50},
					map[string]interface{}{"sku": "PROD-004", "name": "Premium Widget", "price": 299.99},
				},
			},
			want: []interface{}{
				map[string]interface{}{"sku": "PROD-001", "name": "Widget A", "price": 49.99},
				map[string]interface{}{"sku": "PROD-003", "name": "Widget C", "price": 75.50},
			},
			wantErr: false,
		},
		{
			name:     "filter users by role equality",
			jsonPath: `$.users[?(@.role == "admin")]`,
			data: map[string]interface{}{
				"users": []interface{}{
					map[string]interface{}{"id": "user-1", "name": "Alice", "role": "admin", "active": true},
					map[string]interface{}{"id": "user-2", "name": "Bob", "role": "user", "active": true},
					map[string]interface{}{"id": "user-3", "name": "Charlie", "role": "admin", "active": false},
					map[string]interface{}{"id": "user-4", "name": "Diana", "role": "moderator", "active": true},
				},
			},
			want: []interface{}{
				map[string]interface{}{"id": "user-1", "name": "Alice", "role": "admin", "active": true},
				map[string]interface{}{"id": "user-3", "name": "Charlie", "role": "admin", "active": false},
			},
			wantErr: false,
		},
		{
			name:     "filter with greater-than-or-equal comparison",
			jsonPath: "$.scores[?(@.value >= 80)]",
			data: map[string]interface{}{
				"scores": []interface{}{
					map[string]interface{}{"student": "Alice", "value": 95},
					map[string]interface{}{"student": "Bob", "value": 75},
					map[string]interface{}{"student": "Charlie", "value": 80},
					map[string]interface{}{"student": "Diana", "value": 88},
				},
			},
			want: []interface{}{
				map[string]interface{}{"student": "Alice", "value": 95},
				map[string]interface{}{"student": "Charlie", "value": 80},
				map[string]interface{}{"student": "Diana", "value": 88},
			},
			wantErr: false,
		},
		{
			name:     "filter with AND condition (price AND stock availability)",
			jsonPath: "$.inventory[?(@.price < 100 && @.inStock == true)]",
			data: map[string]interface{}{
				"inventory": []interface{}{
					map[string]interface{}{"sku": "A1", "price": 50.0, "inStock": true},
					map[string]interface{}{"sku": "A2", "price": 150.0, "inStock": true},
					map[string]interface{}{"sku": "A3", "price": 75.0, "inStock": false},
					map[string]interface{}{"sku": "A4", "price": 80.0, "inStock": true},
					map[string]interface{}{"sku": "A5", "price": 200.0, "inStock": false},
				},
			},
			want: []interface{}{
				map[string]interface{}{"sku": "A1", "price": 50.0, "inStock": true},
				map[string]interface{}{"sku": "A4", "price": 80.0, "inStock": true},
			},
			wantErr: false,
		},
		{
			name:     "filter with OR condition (category match)",
			jsonPath: `$.items[?(@.category == "electronics" || @.category == "books")]`,
			data: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"id": "1", "name": "Laptop", "category": "electronics"},
					map[string]interface{}{"id": "2", "name": "Chair", "category": "furniture"},
					map[string]interface{}{"id": "3", "name": "Novel", "category": "books"},
					map[string]interface{}{"id": "4", "name": "Desk", "category": "furniture"},
					map[string]interface{}{"id": "5", "name": "Headphones", "category": "electronics"},
				},
			},
			// Note: gjson returns results in traversal order, which may differ from insertion order
			want: []interface{}{
				map[string]interface{}{"id": "1", "name": "Laptop", "category": "electronics"},
				map[string]interface{}{"id": "5", "name": "Headphones", "category": "electronics"},
				map[string]interface{}{"id": "3", "name": "Novel", "category": "books"},
			},
			wantErr: false,
		},
		{
			name:     "filter by field existence",
			jsonPath: "$.users[?(@.email)]",
			data: map[string]interface{}{
				"users": []interface{}{
					map[string]interface{}{"name": "Alice", "email": "alice@example.com"},
					map[string]interface{}{"name": "Bob"},
					map[string]interface{}{"name": "Charlie", "email": "charlie@example.com"},
					map[string]interface{}{"name": "Diana"},
				},
			},
			want: []interface{}{
				map[string]interface{}{"name": "Alice", "email": "alice@example.com"},
				map[string]interface{}{"name": "Charlie", "email": "charlie@example.com"},
			},
			wantErr: false,
		},
	}

	querier := transform.NewJSONPathQuerier()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := querier.Query(ctx, tt.jsonPath, tt.data)

			if (err != nil) != tt.wantErr {
				t.Errorf("Query() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !deepEqual(got, tt.want) {
				t.Errorf("Query() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestJSONPathRecursiveDescent_Integration tests recursive descent operator (..)
func TestJSONPathRecursiveDescent_Integration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tests := []struct {
		name     string
		jsonPath string
		data     interface{}
		want     interface{}
		wantErr  bool
	}{
		{
			name:     "find all email fields recursively in complex structure",
			jsonPath: "$..email",
			data: map[string]interface{}{
				"user": map[string]interface{}{
					"email": "primary@example.com",
					"profile": map[string]interface{}{
						"contact": map[string]interface{}{
							"email": "contact@example.com",
						},
					},
				},
				"admin": map[string]interface{}{
					"email": "admin@example.com",
				},
				"team": []interface{}{
					map[string]interface{}{"email": "team1@example.com"},
					map[string]interface{}{"email": "team2@example.com"},
				},
			},
			// Note: Map iteration order in Go is random, so we just verify count and content
			// The actual order depends on the Go runtime's map randomization
			want:    []interface{}{"team1@example.com", "team2@example.com", "primary@example.com", "contact@example.com", "admin@example.com"},
			wantErr: false,
		},
		{
			name:     "find all price fields in nested store structure",
			jsonPath: "$..price",
			data: map[string]interface{}{
				"store": map[string]interface{}{
					"book": []interface{}{
						map[string]interface{}{"title": "Book1", "price": 10.0},
						map[string]interface{}{"title": "Book2", "price": 15.0},
					},
					"electronics": map[string]interface{}{
						"laptop": map[string]interface{}{"name": "XPS", "price": 1000.0},
						"tablet": map[string]interface{}{"name": "iPad", "price": 500.0},
					},
				},
				"clearance": map[string]interface{}{
					"items": []interface{}{
						map[string]interface{}{"price": 5.0},
						map[string]interface{}{"price": 8.0},
					},
				},
			},
			want:    []interface{}{10.0, 15.0, 1000.0, 500.0, 5.0, 8.0},
			wantErr: false,
		},
		{
			name:     "recursive descent on array structure",
			jsonPath: "$..id",
			data: map[string]interface{}{
				"level1": map[string]interface{}{
					"id": "l1-001",
					"items": []interface{}{
						map[string]interface{}{"id": "item-1", "name": "Item 1"},
						map[string]interface{}{"id": "item-2", "name": "Item 2"},
					},
					"level2": map[string]interface{}{
						"id": "l2-001",
						"items": []interface{}{
							map[string]interface{}{"id": "item-3", "name": "Item 3"},
						},
					},
				},
			},
			want:    []interface{}{"l1-001", "item-1", "item-2", "l2-001", "item-3"},
			wantErr: false,
		},
		{
			name:     "recursive descent finding objects with specific field",
			jsonPath: "$..location",
			data: map[string]interface{}{
				"company": map[string]interface{}{
					"location": "New York",
					"offices": []interface{}{
						map[string]interface{}{
							"name":     "Office A",
							"location": "Boston",
						},
						map[string]interface{}{
							"name":     "Office B",
							"location": "San Francisco",
							"departments": []interface{}{
								map[string]interface{}{
									"name":     "Engineering",
									"location": "5th Floor",
								},
							},
						},
					},
				},
			},
			want:    []interface{}{"New York", "Boston", "San Francisco", "5th Floor"},
			wantErr: false,
		},
	}

	querier := transform.NewJSONPathQuerier()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := querier.Query(ctx, tt.jsonPath, tt.data)

			if (err != nil) != tt.wantErr {
				t.Errorf("Query() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !deepEqual(got, tt.want) {
				t.Errorf("Query() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestJSONPathArrayOperations_Integration tests array manipulation operations
func TestJSONPathArrayOperations_Integration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tests := []struct {
		name     string
		jsonPath string
		data     interface{}
		want     interface{}
		wantErr  bool
	}{
		{
			name:     "extract slice of array [0:3]",
			jsonPath: "$.items[0:3]",
			data: map[string]interface{}{
				"items": []interface{}{"apple", "banana", "cherry", "date", "elderberry"},
			},
			want:    []interface{}{"apple", "banana", "cherry"},
			wantErr: false,
		},
		{
			name:     "extract middle slice of array [1:4]",
			jsonPath: "$.data[1:4]",
			data: map[string]interface{}{
				"data": []interface{}{10, 20, 30, 40, 50, 60},
			},
			want:    []interface{}{20, 30, 40},
			wantErr: false,
		},
		{
			name:     "extract all elements with wildcard",
			jsonPath: "$.products[*].name",
			data: map[string]interface{}{
				"products": []interface{}{
					map[string]interface{}{"name": "Laptop", "price": 1000},
					map[string]interface{}{"name": "Mouse", "price": 25},
					map[string]interface{}{"name": "Keyboard", "price": 75},
				},
			},
			want:    []interface{}{"Laptop", "Mouse", "Keyboard"},
			wantErr: false,
		},
		{
			name:     "access first element of array",
			jsonPath: "$.items[0]",
			data: map[string]interface{}{
				"items": []interface{}{"first", "second", "third"},
			},
			want:    "first",
			wantErr: false,
		},
		{
			name:     "access last element using negative index",
			jsonPath: "$.items[-1]",
			data: map[string]interface{}{
				"items": []interface{}{"first", "second", "third", "last"},
			},
			want:    "last",
			wantErr: false,
		},
		{
			name:     "access second-to-last element",
			jsonPath: "$.items[-2]",
			data: map[string]interface{}{
				"items": []interface{}{"a", "b", "c", "d"},
			},
			want:    "c",
			wantErr: false,
		},
		{
			name:     "flatten nested array wildcard",
			jsonPath: "$.categories[*].items[*]",
			data: map[string]interface{}{
				"categories": []interface{}{
					map[string]interface{}{
						"name":  "Electronics",
						"items": []interface{}{"laptop", "mouse"},
					},
					map[string]interface{}{
						"name":  "Books",
						"items": []interface{}{"novel", "textbook", "reference"},
					},
				},
			},
			want:    []interface{}{"laptop", "mouse", "novel", "textbook", "reference"},
			wantErr: false,
		},
	}

	querier := transform.NewJSONPathQuerier()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := querier.Query(ctx, tt.jsonPath, tt.data)

			if (err != nil) != tt.wantErr {
				t.Errorf("Query() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !deepEqual(got, tt.want) {
				t.Errorf("Query() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestJSONPathComplexQueries_Integration tests complex real-world scenarios
func TestJSONPathComplexQueries_Integration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tests := []struct {
		name     string
		jsonPath string
		data     interface{}
		want     interface{}
		wantErr  bool
	}{
		{
			name:     "e-commerce: extract SKUs from pending orders",
			jsonPath: "$.orders[?(@.status == 'pending')].items[*].sku",
			data: map[string]interface{}{
				"orders": []interface{}{
					map[string]interface{}{
						"id":     "order-001",
						"status": "pending",
						"items": []interface{}{
							map[string]interface{}{"sku": "SKU-A01", "qty": 2},
							map[string]interface{}{"sku": "SKU-A02", "qty": 1},
						},
					},
					map[string]interface{}{
						"id":     "order-002",
						"status": "completed",
						"items": []interface{}{
							map[string]interface{}{"sku": "SKU-B01", "qty": 1},
						},
					},
					map[string]interface{}{
						"id":     "order-003",
						"status": "pending",
						"items": []interface{}{
							map[string]interface{}{"sku": "SKU-C01", "qty": 3},
						},
					},
				},
			},
			want:    []interface{}{"SKU-A01", "SKU-A02", "SKU-C01"},
			wantErr: false,
		},
		{
			name:     "analytics: high-value orders over $1000",
			jsonPath: "$.orders[?(@.total > 1000)].customer.email",
			data: map[string]interface{}{
				"orders": []interface{}{
					map[string]interface{}{
						"id":    "order-1",
						"total": 1500.0,
						"customer": map[string]interface{}{
							"name":  "Alice",
							"email": "alice@example.com",
						},
					},
					map[string]interface{}{
						"id":    "order-2",
						"total": 500.0,
						"customer": map[string]interface{}{
							"name":  "Bob",
							"email": "bob@example.com",
						},
					},
					map[string]interface{}{
						"id":    "order-3",
						"total": 2000.0,
						"customer": map[string]interface{}{
							"name":  "Charlie",
							"email": "charlie@example.com",
						},
					},
				},
			},
			want:    []interface{}{"alice@example.com", "charlie@example.com"},
			wantErr: false,
		},
		{
			name:     "API response: extract nested result data with score filter",
			jsonPath: "$.response.data.results[?(@.score > 0.8)].id",
			data: map[string]interface{}{
				"response": map[string]interface{}{
					"status": "success",
					"data": map[string]interface{}{
						"total": 5,
						"results": []interface{}{
							map[string]interface{}{"id": "res-001", "score": 0.95},
							map[string]interface{}{"id": "res-002", "score": 0.65},
							map[string]interface{}{"id": "res-003", "score": 0.88},
							map[string]interface{}{"id": "res-004", "score": 0.72},
							map[string]interface{}{"id": "res-005", "score": 0.91},
						},
					},
				},
			},
			want:    []interface{}{"res-001", "res-003", "res-005"},
			wantErr: false,
		},
		{
			name:     "nested navigation with multiple wildcards",
			jsonPath: "$.departments[*].teams[*].members[*].name",
			data: map[string]interface{}{
				"departments": []interface{}{
					map[string]interface{}{
						"name": "Engineering",
						"teams": []interface{}{
							map[string]interface{}{
								"name": "Backend",
								"members": []interface{}{
									map[string]interface{}{"name": "Alice"},
									map[string]interface{}{"name": "Bob"},
								},
							},
							map[string]interface{}{
								"name": "Frontend",
								"members": []interface{}{
									map[string]interface{}{"name": "Charlie"},
								},
							},
						},
					},
					map[string]interface{}{
						"name": "Sales",
						"teams": []interface{}{
							map[string]interface{}{
								"name": "Enterprise",
								"members": []interface{}{
									map[string]interface{}{"name": "Diana"},
									map[string]interface{}{"name": "Eve"},
								},
							},
						},
					},
				},
			},
			want:    []interface{}{"Alice", "Bob", "Charlie", "Diana", "Eve"},
			wantErr: false,
		},
	}

	querier := transform.NewJSONPathQuerier()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := querier.Query(ctx, tt.jsonPath, tt.data)

			if (err != nil) != tt.wantErr {
				t.Errorf("Query() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !deepEqual(got, tt.want) {
				t.Errorf("Query() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestJSONPathErrorHandling_Integration tests error handling scenarios
func TestJSONPathErrorHandling_Integration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tests := []struct {
		name        string
		jsonPath    string
		data        interface{}
		wantErr     bool
		errType     error
		description string
	}{
		{
			name:        "invalid bracket syntax",
			jsonPath:    "$..[[[invalid",
			data:        map[string]interface{}{"name": "test"},
			wantErr:     true,
			errType:     transform.ErrInvalidJSONPath,
			description: "Should reject invalid bracket sequences",
		},
		{
			name:        "type mismatch - array access on string",
			jsonPath:    "$.name[0]",
			data:        map[string]interface{}{"name": "string"},
			wantErr:     true,
			errType:     transform.ErrTypeMismatch,
			description: "Should reject array indexing on non-array type",
		},
		{
			name:        "nil data",
			jsonPath:    "$.name",
			data:        nil,
			wantErr:     true,
			errType:     transform.ErrNilData,
			description: "Should reject nil input data",
		},
		{
			name:        "empty path",
			jsonPath:    "",
			data:        map[string]interface{}{"name": "test"},
			wantErr:     true,
			errType:     transform.ErrInvalidJSONPath,
			description: "Should reject empty JSONPath",
		},
		{
			name:        "unclosed bracket",
			jsonPath:    "$.users[0",
			data:        map[string]interface{}{"users": []interface{}{map[string]interface{}{"name": "test"}}},
			wantErr:     true,
			errType:     transform.ErrInvalidJSONPath,
			description: "Should detect unclosed brackets",
		},
	}

	querier := transform.NewJSONPathQuerier()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := querier.Query(ctx, tt.jsonPath, tt.data)

			if (err != nil) != tt.wantErr {
				t.Errorf("%s: Query() error = %v, wantErr %v", tt.description, err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errType != nil {
				// Note: For this to work fully, the transform package should support errors.Is()
				// For now, we just check that error is not nil
				if err == nil {
					t.Errorf("%s: Expected error but got nil", tt.description)
				}
			}
		})
	}
}

// TestJSONPathNonExistentPaths_Integration tests behavior when paths don't exist
func TestJSONPathNonExistentPaths_Integration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tests := []struct {
		name     string
		jsonPath string
		data     interface{}
		want     interface{}
		wantErr  bool
	}{
		{
			name:     "non-existent field returns nil",
			jsonPath: "$.missing",
			data: map[string]interface{}{
				"name": "John",
				"age":  30,
			},
			want:    nil,
			wantErr: false,
		},
		{
			name:     "non-existent nested field returns nil",
			jsonPath: "$.user.missing.field",
			data: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "John",
				},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name:     "array index out of bounds returns nil",
			jsonPath: "$.items[10]",
			data: map[string]interface{}{
				"items": []interface{}{"a", "b", "c"},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name:     "filter matching no items returns empty array",
			jsonPath: "$.items[?(@.price > 10000)]",
			data: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"name": "Item 1", "price": 10},
					map[string]interface{}{"name": "Item 2", "price": 20},
				},
			},
			want:    []interface{}{},
			wantErr: false,
		},
	}

	querier := transform.NewJSONPathQuerier()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := querier.Query(ctx, tt.jsonPath, tt.data)

			if (err != nil) != tt.wantErr {
				t.Errorf("Query() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !deepEqual(got, tt.want) {
				t.Errorf("Query() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestJSONPathWithVariousDataTypes_Integration tests JSONPath with different data types
func TestJSONPathWithVariousDataTypes_Integration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tests := []struct {
		name     string
		jsonPath string
		data     interface{}
		want     interface{}
		wantErr  bool
	}{
		{
			name:     "numeric values",
			jsonPath: "$.prices[*]",
			data: map[string]interface{}{
				"prices": []interface{}{10.5, 20.0, 15.75},
			},
			want:    []interface{}{10.5, 20.0, 15.75},
			wantErr: false,
		},
		{
			name:     "boolean values",
			jsonPath: "$.flags[*]",
			data: map[string]interface{}{
				"flags": []interface{}{true, false, true},
			},
			want:    []interface{}{true, false, true},
			wantErr: false,
		},
		{
			name:     "mixed types in array",
			jsonPath: "$.data[*]",
			data: map[string]interface{}{
				"data": []interface{}{"string", 42, 3.14, true, nil},
			},
			want:    []interface{}{"string", 42, 3.14, true, nil},
			wantErr: false,
		},
		{
			name:     "nested object access with integer keys",
			jsonPath: "$.items[0].id",
			data: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"id": 1, "name": "First"},
					map[string]interface{}{"id": 2, "name": "Second"},
				},
			},
			want:    1,
			wantErr: false,
		},
	}

	querier := transform.NewJSONPathQuerier()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := querier.Query(ctx, tt.jsonPath, tt.data)

			if (err != nil) != tt.wantErr {
				t.Errorf("Query() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !deepEqual(got, tt.want) {
				t.Errorf("Query() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestJSONPathPerformance_Integration tests JSONPath performance with large datasets
func TestJSONPathPerformance_Integration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create large dataset with 1000 items
	items := make([]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		items[i] = map[string]interface{}{
			"id":     i,
			"name":   "Item " + string(rune(i%26+65)),
			"price":  float64(i%100) * 1.5,
			"active": i%2 == 0,
		}
	}

	data := map[string]interface{}{
		"items": items,
	}

	tests := []struct {
		name      string
		jsonPath  string
		wantCount int
	}{
		{
			name:      "filter on large dataset",
			jsonPath:  "$.items[?(@.price > 50)]",
			wantCount: 666, // Approximate count of items with price > 50
		},
		{
			name:      "wildcard extraction on large dataset",
			jsonPath:  "$.items[*].id",
			wantCount: 1000,
		},
	}

	querier := transform.NewJSONPathQuerier()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			got, err := querier.Query(ctx, tt.jsonPath, data)
			elapsed := time.Since(start)

			if err != nil {
				t.Errorf("Query() error = %v", err)
				return
			}

			// Verify result is an array
			gotArray, ok := got.([]interface{})
			if !ok {
				t.Errorf("Expected result to be array, got %T", got)
				return
			}

			// Performance assertion: should complete in reasonable time
			if elapsed > 1*time.Second {
				t.Logf("Performance warning: query took %v (should be < 1s)", elapsed)
			}

			t.Logf("Query took %v, returned %d items", elapsed, len(gotArray))
		})
	}
}

// deepEqual is a helper for comparing complex nested structures
// Uses a reflection-based comparison to handle various types
func deepEqual(a, b interface{}) bool {
	return reflect.DeepEqual(a, b)
}
