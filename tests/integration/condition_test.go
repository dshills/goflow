package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	runtimeexec "github.com/dshills/goflow/pkg/execution"
	"github.com/dshills/goflow/pkg/workflow"
)

// TestConditionNodeEvaluation_SimpleComparison tests basic numeric comparison conditions
func TestConditionNodeEvaluation_SimpleComparison(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tests := []struct {
		name        string
		condition   string
		inputVar    string
		inputValue  interface{}
		expectedRes bool
	}{
		{
			name:        "greater than - true",
			condition:   "fileSize > 1048576",
			inputVar:    "fileSize",
			inputValue:  2097152, // 2MB
			expectedRes: true,
		},
		{
			name:        "greater than - false",
			condition:   "fileSize > 1048576",
			inputVar:    "fileSize",
			inputValue:  524288, // 512KB
			expectedRes: false,
		},
		{
			name:        "less than - true",
			condition:   "count < 10",
			inputVar:    "count",
			inputValue:  5,
			expectedRes: true,
		},
		{
			name:        "less than - false",
			condition:   "count < 10",
			inputVar:    "count",
			inputValue:  15,
			expectedRes: false,
		},
		{
			name:        "equal to - true",
			condition:   `status == "active"`,
			inputVar:    "status",
			inputValue:  "active",
			expectedRes: true,
		},
		{
			name:        "equal to - false",
			condition:   `status == "active"`,
			inputVar:    "status",
			inputValue:  "inactive",
			expectedRes: false,
		},
		{
			name:        "not equal - true",
			condition:   "code != 200",
			inputVar:    "code",
			inputValue:  404,
			expectedRes: true,
		},
		{
			name:        "not equal - false",
			condition:   "code != 200",
			inputVar:    "code",
			inputValue:  200,
			expectedRes: false,
		},
		{
			name:        "greater than or equal - true",
			condition:   "score >= 80",
			inputVar:    "score",
			inputValue:  85,
			expectedRes: true,
		},
		{
			name:        "greater than or equal - boundary",
			condition:   "score >= 80",
			inputVar:    "score",
			inputValue:  80,
			expectedRes: true,
		},
		{
			name:        "less than or equal - true",
			condition:   "price <= 100",
			inputVar:    "price",
			inputValue:  95,
			expectedRes: true,
		},
		{
			name:        "less than or equal - boundary",
			condition:   "price <= 100",
			inputVar:    "price",
			inputValue:  100,
			expectedRes: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml := `
version: "1.0"
name: "condition-simple-test"
variables:
  - name: "` + tt.inputVar + `"
    type: "string"
    default: null
nodes:
  - id: "start"
    type: "start"
  - id: "condition_check"
    type: "condition"
    condition: ` + tt.condition + `
  - id: "true_path"
    type: "passthrough"
  - id: "false_path"
    type: "passthrough"
  - id: "end"
    type: "end"
    return: "completed"
edges:
  - from: "start"
    to: "condition_check"
  - from: "condition_check"
    to: "true_path"
    condition: "true"
  - from: "condition_check"
    to: "false_path"
    condition: "false"
  - from: "true_path"
    to: "end"
  - from: "false_path"
    to: "end"
`

			wf, err := workflow.Parse([]byte(yaml))
			if err != nil {
				t.Fatalf("Failed to parse workflow: %v", err)
			}

			engine := runtimeexec.NewEngine()
			inputs := map[string]interface{}{
				tt.inputVar: tt.inputValue,
			}

			result, err := engine.Execute(ctx, wf, inputs)
			if err != nil {
				t.Fatalf("Workflow execution failed: %v", err)
			}

			if result.Status != execution.StatusCompleted {
				t.Errorf("Expected status %s, got %s", execution.StatusCompleted, result.Status)
			}

			// Verify the correct branch was taken
			// If expectedRes is true, we should have executed true_path
			// If expectedRes is false, we should have executed false_path
			executedNodeIDs := make(map[string]bool)
			for _, nodeExec := range result.NodeExecutions {
				executedNodeIDs[string(nodeExec.NodeID)] = true
			}

			if tt.expectedRes {
				if !executedNodeIDs["true_path"] {
					t.Error("Expected true_path to be executed")
				}
			} else {
				if !executedNodeIDs["false_path"] {
					t.Error("Expected false_path to be executed")
				}
			}
		})
	}
}

// TestConditionNodeEvaluation_MultipleConditions tests conditions with AND/OR operators
func TestConditionNodeEvaluation_MultipleConditions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tests := []struct {
		name        string
		condition   string
		variables   map[string]interface{}
		expectedRes bool
	}{
		{
			name:      "AND operator - both true",
			condition: "status == \"active\" && count > 10",
			variables: map[string]interface{}{
				"status": "active",
				"count":  15,
			},
			expectedRes: true,
		},
		{
			name:      "AND operator - first false",
			condition: "status == \"active\" && count > 10",
			variables: map[string]interface{}{
				"status": "inactive",
				"count":  15,
			},
			expectedRes: false,
		},
		{
			name:      "AND operator - second false",
			condition: "status == \"active\" && count > 10",
			variables: map[string]interface{}{
				"status": "active",
				"count":  5,
			},
			expectedRes: false,
		},
		{
			name:      "OR operator - both true",
			condition: "isEnabled == true || isForced == true",
			variables: map[string]interface{}{
				"isEnabled": true,
				"isForced":  true,
			},
			expectedRes: true,
		},
		{
			name:      "OR operator - first true",
			condition: "isEnabled == true || isForced == true",
			variables: map[string]interface{}{
				"isEnabled": true,
				"isForced":  false,
			},
			expectedRes: true,
		},
		{
			name:      "OR operator - second true",
			condition: "isEnabled == true || isForced == true",
			variables: map[string]interface{}{
				"isEnabled": false,
				"isForced":  true,
			},
			expectedRes: true,
		},
		{
			name:      "OR operator - both false",
			condition: "isEnabled == true || isForced == true",
			variables: map[string]interface{}{
				"isEnabled": false,
				"isForced":  false,
			},
			expectedRes: false,
		},
		{
			name:      "Complex AND/OR - true",
			condition: "(status == \"active\" && count > 10) || priority == \"high\"",
			variables: map[string]interface{}{
				"status":   "inactive",
				"count":    5,
				"priority": "high",
			},
			expectedRes: true,
		},
		{
			name:      "Complex AND/OR - false",
			condition: "(status == \"active\" && count > 10) || priority == \"high\"",
			variables: map[string]interface{}{
				"status":   "inactive",
				"count":    5,
				"priority": "low",
			},
			expectedRes: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			variableYAML := ""
			for name, value := range tt.variables {
				valueStr := ""
				switch v := value.(type) {
				case string:
					valueStr = "\"" + v + "\""
				case bool:
					valueStr = "true"
					if !v {
						valueStr = "false"
					}
				case int:
					valueStr = fmt.Sprintf("%d", v)
				}

				typeStr := "string"
				switch value.(type) {
				case string:
					typeStr = "string"
				case bool:
					typeStr = "boolean"
				case int:
					typeStr = "number"
				}

				variableYAML += `
  - name: "` + name + `"
    type: "` + typeStr + `"
    default: ` + valueStr
			}

			yaml := `
version: "1.0"
name: "condition-multi-test"
variables: ` + variableYAML + `
nodes:
  - id: "start"
    type: "start"
  - id: "condition_check"
    type: "condition"
    condition: '` + tt.condition + `'
  - id: "true_path"
    type: "passthrough"
  - id: "false_path"
    type: "passthrough"
  - id: "end"
    type: "end"
    return: "completed"
edges:
  - from: "start"
    to: "condition_check"
  - from: "condition_check"
    to: "true_path"
    condition: "true"
  - from: "condition_check"
    to: "false_path"
    condition: "false"
  - from: "true_path"
    to: "end"
  - from: "false_path"
    to: "end"
`

			wf, err := workflow.Parse([]byte(yaml))
			if err != nil {
				t.Fatalf("Failed to parse workflow: %v", err)
			}

			engine := runtimeexec.NewEngine()
			result, err := engine.Execute(ctx, wf, tt.variables)
			if err != nil {
				t.Fatalf("Workflow execution failed: %v", err)
			}

			if result.Status != execution.StatusCompleted {
				t.Errorf("Expected status %s, got %s", execution.StatusCompleted, result.Status)
			}

			executedNodeIDs := make(map[string]bool)
			for _, nodeExec := range result.NodeExecutions {
				executedNodeIDs[string(nodeExec.NodeID)] = true
			}

			if tt.expectedRes {
				if !executedNodeIDs["true_path"] {
					t.Error("Expected true_path to be executed")
				}
			} else {
				if !executedNodeIDs["false_path"] {
					t.Error("Expected false_path to be executed")
				}
			}
		})
	}
}

// TestConditionNodeEvaluation_BooleanVariables tests conditions with boolean variables
func TestConditionNodeEvaluation_BooleanVariables(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tests := []struct {
		name        string
		condition   string
		varName     string
		varValue    bool
		expectedRes bool
	}{
		{
			name:        "boolean true comparison - true",
			condition:   "isEnabled == true",
			varName:     "isEnabled",
			varValue:    true,
			expectedRes: true,
		},
		{
			name:        "boolean true comparison - false",
			condition:   "isEnabled == true",
			varName:     "isEnabled",
			varValue:    false,
			expectedRes: false,
		},
		{
			name:        "boolean false comparison - true",
			condition:   "isDisabled == false",
			varName:     "isDisabled",
			varValue:    false,
			expectedRes: true,
		},
		{
			name:        "boolean negation - true",
			condition:   "!isDisabled",
			varName:     "isDisabled",
			varValue:    false,
			expectedRes: true,
		},
		{
			name:        "boolean negation - false",
			condition:   "!isDisabled",
			varName:     "isDisabled",
			varValue:    true,
			expectedRes: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml := `
version: "1.0"
name: "condition-bool-test"
variables:
  - name: "` + tt.varName + `"
    type: "boolean"
    default: false
nodes:
  - id: "start"
    type: "start"
  - id: "condition_check"
    type: "condition"
    condition: '` + tt.condition + `'
  - id: "true_path"
    type: "passthrough"
  - id: "false_path"
    type: "passthrough"
  - id: "end"
    type: "end"
    return: "completed"
edges:
  - from: "start"
    to: "condition_check"
  - from: "condition_check"
    to: "true_path"
    condition: "true"
  - from: "condition_check"
    to: "false_path"
    condition: "false"
  - from: "true_path"
    to: "end"
  - from: "false_path"
    to: "end"
`

			wf, err := workflow.Parse([]byte(yaml))
			if err != nil {
				t.Fatalf("Failed to parse workflow: %v", err)
			}

			engine := runtimeexec.NewEngine()
			inputs := map[string]interface{}{
				tt.varName: tt.varValue,
			}

			result, err := engine.Execute(ctx, wf, inputs)
			if err != nil {
				t.Fatalf("Workflow execution failed: %v", err)
			}

			if result.Status != execution.StatusCompleted {
				t.Errorf("Expected status %s, got %s", execution.StatusCompleted, result.Status)
			}

			executedNodeIDs := make(map[string]bool)
			for _, nodeExec := range result.NodeExecutions {
				executedNodeIDs[string(nodeExec.NodeID)] = true
			}

			if tt.expectedRes {
				if !executedNodeIDs["true_path"] {
					t.Error("Expected true_path to be executed")
				}
			} else {
				if !executedNodeIDs["false_path"] {
					t.Error("Expected false_path to be executed")
				}
			}
		})
	}
}

// TestConditionNodeBranching tests that execution follows the correct branch
func TestConditionNodeBranching_CorrectPathExecution(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	yaml := `
version: "1.0"
name: "branching-test"
variables:
  - name: "userRole"
    type: "string"
    default: "admin"
nodes:
  - id: "start"
    type: "start"
  - id: "check_admin"
    type: "condition"
    condition: "userRole == \"admin\""
  - id: "admin_action"
    type: "passthrough"
  - id: "user_action"
    type: "passthrough"
  - id: "end"
    type: "end"
    return: "completed"
edges:
  - from: "start"
    to: "check_admin"
  - from: "check_admin"
    to: "admin_action"
    condition: "true"
  - from: "check_admin"
    to: "user_action"
    condition: "false"
  - from: "admin_action"
    to: "end"
  - from: "user_action"
    to: "end"
`

	wf, err := workflow.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	engine := runtimeexec.NewEngine()

	// Test 1: Admin branch
	result, err := engine.Execute(ctx, wf, map[string]interface{}{
		"userRole": "admin",
	})
	if err != nil {
		t.Fatalf("Workflow execution failed: %v", err)
	}

	executedNodeIDs := make(map[string]bool)
	for _, nodeExec := range result.NodeExecutions {
		executedNodeIDs[string(nodeExec.NodeID)] = true
	}

	if !executedNodeIDs["admin_action"] {
		t.Error("Expected admin_action to be executed when userRole is admin")
	}

	if executedNodeIDs["user_action"] {
		t.Error("Did not expect user_action to be executed when userRole is admin")
	}

	// Test 2: User branch
	result2, err := engine.Execute(ctx, wf, map[string]interface{}{
		"userRole": "user",
	})
	if err != nil {
		t.Fatalf("Workflow execution failed: %v", err)
	}

	executedNodeIDs2 := make(map[string]bool)
	for _, nodeExec := range result2.NodeExecutions {
		executedNodeIDs2[string(nodeExec.NodeID)] = true
	}

	if !executedNodeIDs2["user_action"] {
		t.Error("Expected user_action to be executed when userRole is user")
	}

	if executedNodeIDs2["admin_action"] {
		t.Error("Did not expect admin_action to be executed when userRole is user")
	}
}

// TestConditionNodeBranching_NestedConditions tests nested conditional branching
func TestConditionNodeBranching_NestedConditions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	yaml := `
version: "1.0"
name: "nested-condition-test"
variables:
  - name: "orderValue"
    type: "number"
    default: 0
  - name: "customerType"
    type: "string"
    default: "regular"
nodes:
  - id: "start"
    type: "start"
  - id: "check_value"
    type: "condition"
    condition: "orderValue > 100"
  - id: "check_customer"
    type: "condition"
    condition: "customerType == \"vip\""
  - id: "vip_premium"
    type: "passthrough"
  - id: "vip_standard"
    type: "passthrough"
  - id: "regular_standard"
    type: "passthrough"
  - id: "end"
    type: "end"
    return: "processing complete"
edges:
  - from: "start"
    to: "check_value"
  - from: "check_value"
    to: "check_customer"
    condition: "true"
  - from: "check_value"
    to: "regular_standard"
    condition: "false"
  - from: "check_customer"
    to: "vip_premium"
    condition: "true"
  - from: "check_customer"
    to: "vip_standard"
    condition: "false"
  - from: "vip_premium"
    to: "end"
  - from: "vip_standard"
    to: "end"
  - from: "regular_standard"
    to: "end"
`

	wf, err := workflow.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	engine := runtimeexec.NewEngine()

	tests := []struct {
		name              string
		orderValue        float64
		customerType      string
		expectedNodeExec  string
		unexpectedNodeExe string
	}{
		{
			name:             "VIP customer with high value",
			orderValue:       200,
			customerType:     "vip",
			expectedNodeExec: "vip_premium",
		},
		{
			name:             "VIP customer with low value",
			orderValue:       50,
			customerType:     "vip",
			expectedNodeExec: "regular_standard",
		},
		{
			name:             "Regular customer with high value",
			orderValue:       200,
			customerType:     "regular",
			expectedNodeExec: "vip_standard",
		},
		{
			name:             "Regular customer with low value",
			orderValue:       50,
			customerType:     "regular",
			expectedNodeExec: "regular_standard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Execute(ctx, wf, map[string]interface{}{
				"orderValue":   tt.orderValue,
				"customerType": tt.customerType,
			})

			if err != nil {
				t.Fatalf("Workflow execution failed: %v", err)
			}

			executedNodeIDs := make(map[string]bool)
			for _, nodeExec := range result.NodeExecutions {
				executedNodeIDs[string(nodeExec.NodeID)] = true
			}

			if !executedNodeIDs[tt.expectedNodeExec] {
				t.Errorf("Expected node %s to be executed, but it was not", tt.expectedNodeExec)
			}
		})
	}
}

// TestConditionNodeErrors_InvalidExpression tests error handling for invalid expressions
func TestConditionNodeErrors_InvalidExpression(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tests := []struct {
		name      string
		condition string
	}{
		{
			name:      "undefined variable",
			condition: "undefinedVar > 10",
		},
		{
			name:      "invalid syntax",
			condition: "value >>>>> 10",
		},
		{
			name:      "unmatched parenthesis",
			condition: "((value > 10)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml := `
version: "1.0"
name: "invalid-condition-test"
variables:
  - name: "value"
    type: "number"
    default: 5
nodes:
  - id: "start"
    type: "start"
  - id: "condition_check"
    type: "condition"
    condition: '` + tt.condition + `'
  - id: "true_path"
    type: "passthrough"
  - id: "false_path"
    type: "passthrough"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "condition_check"
  - from: "condition_check"
    to: "true_path"
    condition: "true"
  - from: "condition_check"
    to: "false_path"
    condition: "false"
  - from: "true_path"
    to: "end"
  - from: "false_path"
    to: "end"
`

			wf, err := workflow.Parse([]byte(yaml))
			if err != nil {
				t.Fatalf("Failed to parse workflow: %v", err)
			}

			engine := runtimeexec.NewEngine()
			result, err := engine.Execute(ctx, wf, nil)

			// Should either error or have a failed status
			if err == nil && result.Status == execution.StatusCompleted {
				t.Error("Expected execution to fail or error for invalid expression")
			}

			// If we got a result, it should have error details
			if result != nil && result.Status == execution.StatusFailed {
				if result.Error == nil {
					t.Error("Expected error details in failed execution")
				}
			}
		})
	}
}

// TestConditionNodeErrors_MissingRequiredVariable tests error when required variable not provided
func TestConditionNodeErrors_MissingRequiredVariable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	yaml := `
version: "1.0"
name: "missing-var-test"
variables:
  - name: "requiredValue"
    type: "number"
    required: true
nodes:
  - id: "start"
    type: "start"
  - id: "condition_check"
    type: "condition"
    condition: "requiredValue > 10"
  - id: "true_path"
    type: "passthrough"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "condition_check"
  - from: "condition_check"
    to: "true_path"
    condition: "true"
  - from: "true_path"
    to: "end"
`

	wf, err := workflow.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	engine := runtimeexec.NewEngine()

	// Execute without providing required variable
	result, err := engine.Execute(ctx, wf, nil)

	// Should error or have failed status
	if err == nil && result.Status == execution.StatusCompleted {
		t.Error("Expected execution to fail when required variable is missing")
	}
}

// TestConditionNodeErrors_TypeMismatch tests error handling for type mismatches
func TestConditionNodeErrors_TypeMismatch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	yaml := `
version: "1.0"
name: "type-mismatch-test"
variables:
  - name: "stringValue"
    type: "string"
    default: "hello"
nodes:
  - id: "start"
    type: "start"
  - id: "condition_check"
    type: "condition"
    condition: "stringValue > 10"
  - id: "true_path"
    type: "passthrough"
  - id: "false_path"
    type: "passthrough"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "condition_check"
  - from: "condition_check"
    to: "true_path"
    condition: "true"
  - from: "condition_check"
    to: "false_path"
    condition: "false"
  - from: "true_path"
    to: "end"
  - from: "false_path"
    to: "end"
`

	wf, err := workflow.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	engine := runtimeexec.NewEngine()
	result, err := engine.Execute(ctx, wf, nil)

	// Should error or have failed status for type mismatch
	if err == nil && result.Status == execution.StatusCompleted {
		t.Error("Expected execution to fail for type mismatch in condition")
	}
}

// TestConditionNodeExecution_ComplexExpressions tests more complex condition expressions
func TestConditionNodeExecution_ComplexExpressions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tests := []struct {
		name        string
		condition   string
		variables   map[string]interface{}
		expectedRes bool
	}{
		{
			name:      "Multiple ANDs",
			condition: "a > 5 && b < 20 && c == \"test\"",
			variables: map[string]interface{}{
				"a": 10,
				"b": 15,
				"c": "test",
			},
			expectedRes: true,
		},
		{
			name:      "Multiple ORs",
			condition: "x == 1 || x == 2 || x == 3",
			variables: map[string]interface{}{
				"x": 2,
			},
			expectedRes: true,
		},
		{
			name:      "Complex parentheses",
			condition: "(a > 10 && b < 20) || (c == \"admin\" && d == true)",
			variables: map[string]interface{}{
				"a": 5,
				"b": 25,
				"c": "admin",
				"d": true,
			},
			expectedRes: true,
		},
		{
			name:      "Negation with AND",
			condition: "!isDeleted && status == \"active\"",
			variables: map[string]interface{}{
				"isDeleted": false,
				"status":    "active",
			},
			expectedRes: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			variableYAML := ""
			for name, value := range tt.variables {
				valueStr := ""
				switch v := value.(type) {
				case string:
					valueStr = "\"" + v + "\""
				case bool:
					valueStr = "true"
					if !v {
						valueStr = "false"
					}
				case int:
					valueStr = fmt.Sprintf("%d", v)
				case float64:
					valueStr = fmt.Sprintf("%f", v)
				}

				typeStr := "string"
				switch value.(type) {
				case string:
					typeStr = "string"
				case bool:
					typeStr = "boolean"
				case int:
					typeStr = "number"
				}

				variableYAML += `
  - name: "` + name + `"
    type: "` + typeStr + `"
    default: ` + valueStr
			}

			yaml := `
version: "1.0"
name: "complex-condition-test"
variables: ` + variableYAML + `
nodes:
  - id: "start"
    type: "start"
  - id: "condition_check"
    type: "condition"
    condition: '` + tt.condition + `'
  - id: "true_path"
    type: "passthrough"
  - id: "false_path"
    type: "passthrough"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "condition_check"
  - from: "condition_check"
    to: "true_path"
    condition: "true"
  - from: "condition_check"
    to: "false_path"
    condition: "false"
  - from: "true_path"
    to: "end"
  - from: "false_path"
    to: "end"
`

			wf, err := workflow.Parse([]byte(yaml))
			if err != nil {
				t.Fatalf("Failed to parse workflow: %v", err)
			}

			engine := runtimeexec.NewEngine()
			result, err := engine.Execute(ctx, wf, tt.variables)
			if err != nil {
				t.Fatalf("Workflow execution failed: %v", err)
			}

			executedNodeIDs := make(map[string]bool)
			for _, nodeExec := range result.NodeExecutions {
				executedNodeIDs[string(nodeExec.NodeID)] = true
			}

			if tt.expectedRes {
				if !executedNodeIDs["true_path"] {
					t.Error("Expected true_path to be executed")
				}
			} else {
				if !executedNodeIDs["false_path"] {
					t.Error("Expected false_path to be executed")
				}
			}
		})
	}
}

// TestConditionNodeExecution_ExpressionEvaluationOrder tests evaluation precedence
func TestConditionNodeExecution_ExpressionEvaluationOrder(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test that AND has higher precedence than OR: a || b && c == (a || (b && c))
	yaml := `
version: "1.0"
name: "precedence-test"
variables:
  - name: "a"
    type: "boolean"
    default: false
  - name: "b"
    type: "boolean"
    default: false
  - name: "c"
    type: "boolean"
    default: true
nodes:
  - id: "start"
    type: "start"
  - id: "condition_check"
    type: "condition"
    condition: "a || b && c"
  - id: "true_path"
    type: "passthrough"
  - id: "false_path"
    type: "passthrough"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "condition_check"
  - from: "condition_check"
    to: "true_path"
    condition: "true"
  - from: "condition_check"
    to: "false_path"
    condition: "false"
  - from: "true_path"
    to: "end"
  - from: "false_path"
    to: "end"
`

	wf, err := workflow.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	engine := runtimeexec.NewEngine()

	// When a=false, b=false, c=true:
	// a || b && c should be: false || (false && true) = false || false = false
	result, err := engine.Execute(ctx, wf, map[string]interface{}{
		"a": false,
		"b": false,
		"c": true,
	})

	if err != nil {
		t.Fatalf("Workflow execution failed: %v", err)
	}

	executedNodeIDs := make(map[string]bool)
	for _, nodeExec := range result.NodeExecutions {
		executedNodeIDs[string(nodeExec.NodeID)] = true
	}

	// Should evaluate to false (false_path should be taken)
	if !executedNodeIDs["false_path"] {
		t.Error("Expected false_path due to operator precedence")
	}
}
