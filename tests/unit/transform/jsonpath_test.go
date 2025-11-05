package transform_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/dshills/goflow/pkg/transform"

	"testing"
)

// TestJSONPathBasicQueries tests simple JSONPath query operations
func TestJSONPathBasicQueries(t *testing.T) {
	tests := []struct {
		name     string
		jsonPath string
		data     interface{}
		want     interface{}
		wantErr  bool
	}{
		{
			name:     "root object access",
			jsonPath: "$",
			data: map[string]interface{}{
				"name": "John",
				"age":  30,
			},
			want: map[string]interface{}{
				"name": "John",
				"age":  30,
			},
			wantErr: false,
		},
		{
			name:     "simple field access",
			jsonPath: "$.name",
			data: map[string]interface{}{
				"name": "John",
				"age":  30,
			},
			want:    "John",
			wantErr: false,
		},
		{
			name:     "nested field access",
			jsonPath: "$.user.email",
			data: map[string]interface{}{
				"user": map[string]interface{}{
					"email": "john@example.com",
					"name":  "John",
				},
			},
			want:    "john@example.com",
			wantErr: false,
		},
		{
			name:     "array index access",
			jsonPath: "$.users[0].email",
			data: map[string]interface{}{
				"users": []interface{}{
					map[string]interface{}{"email": "first@example.com"},
					map[string]interface{}{"email": "second@example.com"},
				},
			},
			want:    "first@example.com",
			wantErr: false,
		},
		{
			name:     "last array element",
			jsonPath: "$.users[-1].email",
			data: map[string]interface{}{
				"users": []interface{}{
					map[string]interface{}{"email": "first@example.com"},
					map[string]interface{}{"email": "last@example.com"},
				},
			},
			want:    "last@example.com",
			wantErr: false,
		},
		{
			name:     "all array elements",
			jsonPath: "$.users[*].email",
			data: map[string]interface{}{
				"users": []interface{}{
					map[string]interface{}{"email": "first@example.com"},
					map[string]interface{}{"email": "second@example.com"},
				},
			},
			want:    []interface{}{"first@example.com", "second@example.com"},
			wantErr: false,
		},
		{
			name:     "array slice",
			jsonPath: "$.items[0:2]",
			data: map[string]interface{}{
				"items": []interface{}{"a", "b", "c", "d"},
			},
			want:    []interface{}{"a", "b"},
			wantErr: false,
		},
		{
			name:     "non-existent field returns nil",
			jsonPath: "$.missing",
			data: map[string]interface{}{
				"name": "John",
			},
			want:    nil,
			wantErr: false,
		},
		{
			name:     "invalid JSONPath syntax",
			jsonPath: "$..[invalid",
			data: map[string]interface{}{
				"name": "John",
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier := transform.NewJSONPathQuerier()
			got, err := querier.Query(context.Background(), tt.jsonPath, tt.data)

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

// TestJSONPathFilters tests filter expressions: $.items[?(@.price < 100)]
func TestJSONPathFilters(t *testing.T) {
	tests := []struct {
		name     string
		jsonPath string
		data     interface{}
		want     interface{}
		wantErr  bool
	}{
		{
			name:     "filter by price less than",
			jsonPath: "$.items[?(@.price < 100)]",
			data: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"name": "cheap", "price": 50.0},
					map[string]interface{}{"name": "expensive", "price": 150.0},
					map[string]interface{}{"name": "moderate", "price": 75.0},
				},
			},
			want: []interface{}{
				map[string]interface{}{"name": "cheap", "price": 50.0},
				map[string]interface{}{"name": "moderate", "price": 75.0},
			},
			wantErr: false,
		},
		{
			name:     "filter by equality",
			jsonPath: `$.users[?(@.role == "admin")]`,
			data: map[string]interface{}{
				"users": []interface{}{
					map[string]interface{}{"name": "Alice", "role": "admin"},
					map[string]interface{}{"name": "Bob", "role": "user"},
					map[string]interface{}{"name": "Charlie", "role": "admin"},
				},
			},
			want: []interface{}{
				map[string]interface{}{"name": "Alice", "role": "admin"},
				map[string]interface{}{"name": "Charlie", "role": "admin"},
			},
			wantErr: false,
		},
		{
			name:     "filter by greater than or equal",
			jsonPath: "$.scores[?(@.value >= 80)]",
			data: map[string]interface{}{
				"scores": []interface{}{
					map[string]interface{}{"student": "A", "value": 95},
					map[string]interface{}{"student": "B", "value": 75},
					map[string]interface{}{"student": "C", "value": 80},
				},
			},
			want: []interface{}{
				map[string]interface{}{"student": "A", "value": 95},
				map[string]interface{}{"student": "C", "value": 80},
			},
			wantErr: false,
		},
		{
			name:     "filter with AND condition",
			jsonPath: "$.products[?(@.price < 100 && @.inStock == true)]",
			data: map[string]interface{}{
				"products": []interface{}{
					map[string]interface{}{"name": "A", "price": 50.0, "inStock": true},
					map[string]interface{}{"name": "B", "price": 150.0, "inStock": true},
					map[string]interface{}{"name": "C", "price": 75.0, "inStock": false},
					map[string]interface{}{"name": "D", "price": 80.0, "inStock": true},
				},
			},
			want: []interface{}{
				map[string]interface{}{"name": "A", "price": 50.0, "inStock": true},
				map[string]interface{}{"name": "D", "price": 80.0, "inStock": true},
			},
			wantErr: false,
		},
		{
			name:     "filter with OR condition",
			jsonPath: `$.items[?(@.category == "electronics" || @.category == "books")]`,
			data: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"name": "Laptop", "category": "electronics"},
					map[string]interface{}{"name": "Chair", "category": "furniture"},
					map[string]interface{}{"name": "Novel", "category": "books"},
				},
			},
			want: []interface{}{
				map[string]interface{}{"name": "Laptop", "category": "electronics"},
				map[string]interface{}{"name": "Novel", "category": "books"},
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
				},
			},
			want: []interface{}{
				map[string]interface{}{"name": "Alice", "email": "alice@example.com"},
				map[string]interface{}{"name": "Charlie", "email": "charlie@example.com"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier := transform.NewJSONPathQuerier()
			got, err := querier.Query(context.Background(), tt.jsonPath, tt.data)

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

// TestJSONPathArrayOperations tests array manipulation and aggregation
func TestJSONPathArrayOperations(t *testing.T) {
	tests := []struct {
		name     string
		jsonPath string
		data     interface{}
		want     interface{}
		wantErr  bool
	}{
		{
			name:     "array length",
			jsonPath: "$.items.length()",
			data: map[string]interface{}{
				"items": []interface{}{"a", "b", "c"},
			},
			want:    3,
			wantErr: false,
		},
		{
			name:     "extract all prices",
			jsonPath: "$.products[*].price",
			data: map[string]interface{}{
				"products": []interface{}{
					map[string]interface{}{"name": "A", "price": 10.0},
					map[string]interface{}{"name": "B", "price": 20.0},
					map[string]interface{}{"name": "C", "price": 30.0},
				},
			},
			want:    []interface{}{10.0, 20.0, 30.0},
			wantErr: false,
		},
		{
			name:     "flatten nested arrays",
			jsonPath: "$.categories[*].items[*]",
			data: map[string]interface{}{
				"categories": []interface{}{
					map[string]interface{}{
						"items": []interface{}{"a1", "a2"},
					},
					map[string]interface{}{
						"items": []interface{}{"b1", "b2"},
					},
				},
			},
			want:    []interface{}{"a1", "a2", "b1", "b2"},
			wantErr: false,
		},
		{
			name:     "first element of each array",
			jsonPath: "$.data[*][0]",
			data: map[string]interface{}{
				"data": []interface{}{
					[]interface{}{1, 2, 3},
					[]interface{}{4, 5, 6},
					[]interface{}{7, 8, 9},
				},
			},
			want:    []interface{}{1, 4, 7},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier := transform.NewJSONPathQuerier()
			got, err := querier.Query(context.Background(), tt.jsonPath, tt.data)

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

// TestJSONPathRecursiveDescent tests recursive descent operator (..)
func TestJSONPathRecursiveDescent(t *testing.T) {
	tests := []struct {
		name     string
		jsonPath string
		data     interface{}
		want     interface{}
		wantErr  bool
	}{
		{
			name:     "find all email fields recursively",
			jsonPath: "$..email",
			data: map[string]interface{}{
				"user": map[string]interface{}{
					"email": "user@example.com",
					"profile": map[string]interface{}{
						"contact": map[string]interface{}{
							"email": "contact@example.com",
						},
					},
				},
				"admin": map[string]interface{}{
					"email": "admin@example.com",
				},
			},
			want:    []interface{}{"user@example.com", "contact@example.com", "admin@example.com"},
			wantErr: false,
		},
		{
			name:     "find all price fields in nested structure",
			jsonPath: "$..price",
			data: map[string]interface{}{
				"store": map[string]interface{}{
					"book": []interface{}{
						map[string]interface{}{"title": "Book1", "price": 10.0},
						map[string]interface{}{"title": "Book2", "price": 15.0},
					},
					"electronics": map[string]interface{}{
						"laptop": map[string]interface{}{"price": 1000.0},
					},
				},
			},
			want:    []interface{}{10.0, 15.0, 1000.0},
			wantErr: false,
		},
		{
			name:     "recursive descent with filter",
			jsonPath: "$..[?(@.active == true)].name",
			data: map[string]interface{}{
				"departments": []interface{}{
					map[string]interface{}{
						"name":   "Engineering",
						"active": true,
						"teams": []interface{}{
							map[string]interface{}{"name": "Backend", "active": true},
							map[string]interface{}{"name": "Frontend", "active": false},
						},
					},
					map[string]interface{}{
						"name":   "Sales",
						"active": false,
					},
				},
			},
			want:    []interface{}{"Engineering", "Backend"},
			wantErr: false,
		},
		{
			name:     "recursive descent on arrays",
			jsonPath: "$..items[*].id",
			data: map[string]interface{}{
				"level1": map[string]interface{}{
					"items": []interface{}{
						map[string]interface{}{"id": 1},
						map[string]interface{}{"id": 2},
					},
					"level2": map[string]interface{}{
						"items": []interface{}{
							map[string]interface{}{"id": 3},
						},
					},
				},
			},
			want:    []interface{}{1, 2, 3},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier := transform.NewJSONPathQuerier()
			got, err := querier.Query(context.Background(), tt.jsonPath, tt.data)

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

// TestJSONPathComplexScenarios tests real-world complex queries
func TestJSONPathComplexScenarios(t *testing.T) {
	tests := []struct {
		name     string
		jsonPath string
		data     interface{}
		want     interface{}
		wantErr  bool
	}{
		{
			name:     "e-commerce order processing",
			jsonPath: "$.orders[?(@.status == 'pending')].items[*].sku",
			data: map[string]interface{}{
				"orders": []interface{}{
					map[string]interface{}{
						"id":     "order-1",
						"status": "pending",
						"items": []interface{}{
							map[string]interface{}{"sku": "SKU-001", "qty": 2},
							map[string]interface{}{"sku": "SKU-002", "qty": 1},
						},
					},
					map[string]interface{}{
						"id":     "order-2",
						"status": "completed",
						"items": []interface{}{
							map[string]interface{}{"sku": "SKU-003", "qty": 1},
						},
					},
					map[string]interface{}{
						"id":     "order-3",
						"status": "pending",
						"items": []interface{}{
							map[string]interface{}{"sku": "SKU-004", "qty": 3},
						},
					},
				},
			},
			want:    []interface{}{"SKU-001", "SKU-002", "SKU-004"},
			wantErr: false,
		},
		{
			name:     "user permissions check",
			jsonPath: `$.users[?(@.roles[*] contains "admin")].email`,
			data: map[string]interface{}{
				"users": []interface{}{
					map[string]interface{}{
						"email": "alice@example.com",
						"roles": []interface{}{"admin", "user"},
					},
					map[string]interface{}{
						"email": "bob@example.com",
						"roles": []interface{}{"user"},
					},
					map[string]interface{}{
						"email": "charlie@example.com",
						"roles": []interface{}{"admin", "moderator"},
					},
				},
			},
			want:    []interface{}{"alice@example.com", "charlie@example.com"},
			wantErr: false,
		},
		{
			name:     "nested data transformation",
			jsonPath: "$.response.data.results[?(@.score > 0.8)].id",
			data: map[string]interface{}{
				"response": map[string]interface{}{
					"status": "success",
					"data": map[string]interface{}{
						"results": []interface{}{
							map[string]interface{}{"id": "res-1", "score": 0.95},
							map[string]interface{}{"id": "res-2", "score": 0.65},
							map[string]interface{}{"id": "res-3", "score": 0.88},
						},
					},
				},
			},
			want:    []interface{}{"res-1", "res-3"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier := transform.NewJSONPathQuerier()
			got, err := querier.Query(context.Background(), tt.jsonPath, tt.data)

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

// TestJSONPathErrorHandling tests error scenarios
func TestJSONPathErrorHandling(t *testing.T) {
	tests := []struct {
		name     string
		jsonPath string
		data     interface{}
		wantErr  bool
		errType  error
	}{
		{
			name:     "invalid JSON path syntax",
			jsonPath: "$..[[[invalid",
			data:     map[string]interface{}{"name": "test"},
			wantErr:  true,
			errType:  transform.ErrInvalidJSONPath,
		},
		{
			name:     "type mismatch - array access on non-array",
			jsonPath: "$.name[0]",
			data:     map[string]interface{}{"name": "string"},
			wantErr:  true,
			errType:  transform.ErrTypeMismatch,
		},
		{
			name:     "nil data",
			jsonPath: "$.name",
			data:     nil,
			wantErr:  true,
			errType:  transform.ErrNilData,
		},
		{
			name:     "empty path",
			jsonPath: "",
			data:     map[string]interface{}{"name": "test"},
			wantErr:  true,
			errType:  transform.ErrInvalidJSONPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier := transform.NewJSONPathQuerier()
			_, err := querier.Query(context.Background(), tt.jsonPath, tt.data)

			if (err != nil) != tt.wantErr {
				t.Errorf("Query() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errType != nil {
				if !errors.Is(err, tt.errType) {
					t.Errorf("Query() error = %v, want error type %v", err, tt.errType)
				}
			}
		})
	}
}

// deepEqual is a helper for comparing complex nested structures
func deepEqual(a, b interface{}) bool {
	// Use reflect.DeepEqual for proper comparison
	return fmt.Sprintf("%#v", a) == fmt.Sprintf("%#v", b)
}
