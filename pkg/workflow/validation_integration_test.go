package workflow

import (
	"testing"
)

// TestWorkflowValidation_EndToEnd tests the full validation pipeline
// from workflow construction through validation
func TestWorkflowValidation_EndToEnd(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func() (*Workflow, error)
		wantErr     bool
		errContains string
	}{
		{
			name: "valid workflow with condition and transform",
			setupFunc: func() (*Workflow, error) {
				wf, err := NewWorkflow("test", "Test workflow")
				if err != nil {
					return nil, err
				}

				// Add variables
				wf.AddVariable(&Variable{Name: "count", Type: "number", DefaultValue: 10})
				wf.AddVariable(&Variable{Name: "data", Type: "object"})
				wf.AddVariable(&Variable{Name: "result", Type: "string"})

				// Add nodes
				start := &StartNode{ID: "start"}
				condition := &ConditionNode{ID: "cond1", Condition: "count > 5"}
				transform := &TransformNode{
					ID:             "trans1",
					InputVariable:  "data",
					Expression:     "$.users[0].name",
					OutputVariable: "result",
				}
				end := &EndNode{ID: "end"}

				wf.AddNode(start)
				wf.AddNode(condition)
				wf.AddNode(transform)
				wf.AddNode(end)

				// Add edges
				wf.AddEdge(&Edge{FromNodeID: "start", ToNodeID: "cond1"})
				wf.AddEdge(&Edge{FromNodeID: "cond1", ToNodeID: "trans1", Condition: "true"})
				wf.AddEdge(&Edge{FromNodeID: "cond1", ToNodeID: "end", Condition: "false"})
				wf.AddEdge(&Edge{FromNodeID: "trans1", ToNodeID: "end"})

				return wf, nil
			},
			wantErr: false,
		},
		{
			name: "invalid - condition references undefined variable",
			setupFunc: func() (*Workflow, error) {
				wf, err := NewWorkflow("test", "Test workflow")
				if err != nil {
					return nil, err
				}

				wf.AddVariable(&Variable{Name: "count", Type: "number"})

				start := &StartNode{ID: "start"}
				// Reference 'price' which doesn't exist
				condition := &ConditionNode{ID: "cond1", Condition: "price > 100"}
				end1 := &EndNode{ID: "end1"}
				end2 := &EndNode{ID: "end2"}

				wf.AddNode(start)
				wf.AddNode(condition)
				wf.AddNode(end1)
				wf.AddNode(end2)

				wf.AddEdge(&Edge{FromNodeID: "start", ToNodeID: "cond1"})
				wf.AddEdge(&Edge{FromNodeID: "cond1", ToNodeID: "end1", Condition: "true"})
				wf.AddEdge(&Edge{FromNodeID: "cond1", ToNodeID: "end2", Condition: "false"})

				return wf, nil
			},
			wantErr:     true,
			errContains: "undefined variable",
		},
		{
			name: "invalid - transform with invalid JSONPath",
			setupFunc: func() (*Workflow, error) {
				wf, err := NewWorkflow("test", "Test workflow")
				if err != nil {
					return nil, err
				}

				wf.AddVariable(&Variable{Name: "data", Type: "object"})
				wf.AddVariable(&Variable{Name: "result", Type: "string"})

				start := &StartNode{ID: "start"}
				// Invalid JSONPath - unclosed bracket
				transform := &TransformNode{
					ID:             "trans1",
					InputVariable:  "data",
					Expression:     "$.users[0.name",
					OutputVariable: "result",
				}
				end := &EndNode{ID: "end"}

				wf.AddNode(start)
				wf.AddNode(transform)
				wf.AddNode(end)

				wf.AddEdge(&Edge{FromNodeID: "start", ToNodeID: "trans1"})
				wf.AddEdge(&Edge{FromNodeID: "trans1", ToNodeID: "end"})

				return wf, nil
			},
			wantErr:     true,
			errContains: "invalid JSONPath",
		},
		{
			name: "invalid - template with undefined variable",
			setupFunc: func() (*Workflow, error) {
				wf, err := NewWorkflow("test", "Test workflow")
				if err != nil {
					return nil, err
				}

				wf.AddVariable(&Variable{Name: "data", Type: "object"})
				wf.AddVariable(&Variable{Name: "result", Type: "string"})

				start := &StartNode{ID: "start"}
				// Template references 'userName' which doesn't exist
				transform := &TransformNode{
					ID:             "trans1",
					InputVariable:  "data",
					Expression:     "Hello ${userName}",
					OutputVariable: "result",
				}
				end := &EndNode{ID: "end"}

				wf.AddNode(start)
				wf.AddNode(transform)
				wf.AddNode(end)

				wf.AddEdge(&Edge{FromNodeID: "start", ToNodeID: "trans1"})
				wf.AddEdge(&Edge{FromNodeID: "trans1", ToNodeID: "end"})

				return wf, nil
			},
			wantErr:     true,
			errContains: "undefined variable in template",
		},
		{
			name: "invalid - condition with unsafe operation",
			setupFunc: func() (*Workflow, error) {
				wf, err := NewWorkflow("test", "Test workflow")
				if err != nil {
					return nil, err
				}

				wf.AddVariable(&Variable{Name: "file", Type: "string"})

				start := &StartNode{ID: "start"}
				// Unsafe operation
				condition := &ConditionNode{ID: "cond1", Condition: "os.ReadFile(file)"}
				end1 := &EndNode{ID: "end1"}
				end2 := &EndNode{ID: "end2"}

				wf.AddNode(start)
				wf.AddNode(condition)
				wf.AddNode(end1)
				wf.AddNode(end2)

				wf.AddEdge(&Edge{FromNodeID: "start", ToNodeID: "cond1"})
				wf.AddEdge(&Edge{FromNodeID: "cond1", ToNodeID: "end1", Condition: "true"})
				wf.AddEdge(&Edge{FromNodeID: "cond1", ToNodeID: "end2", Condition: "false"})

				return wf, nil
			},
			wantErr:     true,
			errContains: "unsafe operation",
		},
		{
			name: "valid - template with nested variable access",
			setupFunc: func() (*Workflow, error) {
				wf, err := NewWorkflow("test", "Test workflow")
				if err != nil {
					return nil, err
				}

				wf.AddVariable(&Variable{Name: "user", Type: "object"})
				wf.AddVariable(&Variable{Name: "greeting", Type: "string"})

				start := &StartNode{ID: "start"}
				// Template with nested access - should extract "user" as base variable
				transform := &TransformNode{
					ID:             "trans1",
					InputVariable:  "user",
					Expression:     "Hello ${user.name}",
					OutputVariable: "greeting",
				}
				end := &EndNode{ID: "end"}

				wf.AddNode(start)
				wf.AddNode(transform)
				wf.AddNode(end)

				wf.AddEdge(&Edge{FromNodeID: "start", ToNodeID: "trans1"})
				wf.AddEdge(&Edge{FromNodeID: "trans1", ToNodeID: "end"})

				return wf, nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf, err := tt.setupFunc()
			if err != nil {
				t.Fatalf("Failed to set up workflow: %v", err)
			}

			err = wf.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Workflow.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil && tt.errContains != "" {
				if !containsString(err.Error(), tt.errContains) {
					t.Errorf("Workflow.Validate() error = %v, should contain %q", err, tt.errContains)
				}
			}
		})
	}
}

// TestExpressionValidation_RealWorldExamples tests validation with real-world expression patterns
func TestExpressionValidation_RealWorldExamples(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		variables  []*Variable
		wantErr    bool
	}{
		{
			name:       "price comparison with multiple conditions",
			expression: "price > 100 && quantity < 50 && inStock == true",
			variables: []*Variable{
				{Name: "price", Type: "number"},
				{Name: "quantity", Type: "number"},
				{Name: "inStock", Type: "boolean"},
			},
			wantErr: false,
		},
		{
			name:       "email validation pattern",
			expression: "email contains '@' && email contains '.'",
			variables: []*Variable{
				{Name: "email", Type: "string"},
			},
			wantErr: false,
		},
		{
			name:       "complex nested condition",
			expression: "(age >= 18 && age <= 65) || verified == true",
			variables: []*Variable{
				{Name: "age", Type: "number"},
				{Name: "verified", Type: "boolean"},
			},
			wantErr: false,
		},
		{
			name:       "status check with OR",
			expression: "status == 'active' || status == 'pending'",
			variables: []*Variable{
				{Name: "status", Type: "string"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf, _ := NewWorkflow("test", "test")
			for _, v := range tt.variables {
				wf.AddVariable(v)
			}

			node := &ConditionNode{
				ID:        "cond1",
				Condition: tt.expression,
			}

			err := wf.validateConditionExpression(node)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConditionExpression() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
