package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/dshills/goflow/pkg/workflow"
)

// TestTemplateInstantiationWithAllParameters tests creating a workflow from a template
// with all required and optional parameters provided
func TestTemplateInstantiationWithAllParameters(t *testing.T) {
	// Define a workflow template with parameter placeholders
	template := &workflow.WorkflowTemplate{
		Name:        "api-integration-template",
		Description: "Template for integrating with external APIs",
		Version:     "1.0.0",
		Parameters: []workflow.TemplateParameter{
			{
				Name:        "apiEndpoint",
				Type:        workflow.ParameterTypeString,
				Required:    true,
				Description: "The API endpoint URL",
			},
			{
				Name:        "apiKey",
				Type:        workflow.ParameterTypeString,
				Required:    true,
				Description: "API authentication key",
			},
			{
				Name:        "timeout",
				Type:        workflow.ParameterTypeNumber,
				Required:    false,
				Default:     30,
				Description: "Request timeout in seconds",
			},
			{
				Name:        "retryEnabled",
				Type:        workflow.ParameterTypeBoolean,
				Required:    false,
				Default:     true,
				Description: "Enable automatic retry on failure",
			},
		},
		WorkflowSpec: workflow.WorkflowSpec{
			Nodes: []workflow.NodeSpec{
				{
					ID:   "start",
					Type: "start",
				},
				{
					ID:   "fetch_data",
					Type: "mcp_tool",
					Config: map[string]interface{}{
						"server": "fetch-server",
						"tool":   "fetch",
						"parameters": map[string]interface{}{
							"url":     "{{apiEndpoint}}",
							"timeout": "{{timeout}}",
						},
					},
				},
			},
			Edges: []workflow.EdgeSpec{
				{
					From: "start",
					To:   "fetch_data",
				},
			},
		},
	}

	// Instantiate template with parameters
	params := map[string]interface{}{
		"apiEndpoint":  "https://api.example.com/v1/data",
		"apiKey":       "secret-key-123",
		"timeout":      60,
		"retryEnabled": false,
	}

	wf, err := workflow.InstantiateTemplate(context.Background(), template, params)
	if err != nil {
		t.Fatalf("InstantiateTemplate() failed: %v", err)
	}

	// Verify workflow was created
	if wf == nil {
		t.Fatal("Expected workflow to be created")
	}

	// Verify parameters were substituted
	if wf.Name != "api-integration-template" {
		t.Errorf("Expected workflow name 'api-integration-template', got %q", wf.Name)
	}

	// Verify node configuration has substituted values
	fetchNode := findNodeByID(wf.Nodes, "fetch_data")
	if fetchNode == nil {
		t.Fatal("Expected fetch_data node to exist")
	}

	nodeConfig := fetchNode.GetConfiguration()
	params, ok := nodeConfig["parameters"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected parameters in node config")
	}

	if params["url"] != "https://api.example.com/v1/data" {
		t.Errorf("Expected url to be substituted, got %v", params["url"])
	}

	if params["timeout"] != 60 {
		t.Errorf("Expected timeout to be 60, got %v", params["timeout"])
	}
}

// TestTemplateInstantiationMissingRequiredParameters tests that instantiation fails
// when required parameters are not provided
func TestTemplateInstantiationMissingRequiredParameters(t *testing.T) {
	template := &workflow.WorkflowTemplate{
		Name:        "test-template",
		Description: "Template requiring parameters",
		Version:     "1.0.0",
		Parameters: []workflow.TemplateParameter{
			{
				Name:        "requiredParam1",
				Type:        workflow.ParameterTypeString,
				Required:    true,
				Description: "A required string parameter",
			},
			{
				Name:        "requiredParam2",
				Type:        workflow.ParameterTypeNumber,
				Required:    true,
				Description: "A required number parameter",
			},
			{
				Name:        "optionalParam",
				Type:        workflow.ParameterTypeString,
				Required:    false,
				Default:     "default-value",
				Description: "An optional parameter",
			},
		},
		WorkflowSpec: workflow.WorkflowSpec{
			Nodes: []workflow.NodeSpec{
				{
					ID:   "start",
					Type: "start",
				},
			},
		},
	}

	// Missing requiredParam2
	params := map[string]interface{}{
		"requiredParam1": "value1",
	}

	_, err := workflow.InstantiateTemplate(context.Background(), template, params)
	if err == nil {
		t.Fatal("Expected error for missing required parameter")
	}

	if !errors.Is(err, workflow.ErrMissingRequiredParameter) {
		t.Errorf("Expected ErrMissingRequiredParameter, got %v", err)
	}

	// Verify error message mentions the missing parameter
	if err.Error() == "" {
		t.Error("Expected error message to be non-empty")
	}
}

// TestTemplateInstantiationWithDefaults tests that default values are used
// when optional parameters are not provided
func TestTemplateInstantiationWithDefaults(t *testing.T) {
	template := &workflow.WorkflowTemplate{
		Name:        "template-with-defaults",
		Description: "Template with default parameter values",
		Version:     "1.0.0",
		Parameters: []workflow.TemplateParameter{
			{
				Name:     "requiredParam",
				Type:     workflow.ParameterTypeString,
				Required: true,
			},
			{
				Name:     "optionalString",
				Type:     workflow.ParameterTypeString,
				Required: false,
				Default:  "default-string",
			},
			{
				Name:     "optionalNumber",
				Type:     workflow.ParameterTypeNumber,
				Required: false,
				Default:  42,
			},
			{
				Name:     "optionalBoolean",
				Type:     workflow.ParameterTypeBoolean,
				Required: false,
				Default:  true,
			},
			{
				Name:     "optionalArray",
				Type:     workflow.ParameterTypeArray,
				Required: false,
				Default:  []interface{}{"item1", "item2"},
			},
		},
		WorkflowSpec: workflow.WorkflowSpec{
			Nodes: []workflow.NodeSpec{
				{
					ID:   "node1",
					Type: "transform",
					Config: map[string]interface{}{
						"value1": "{{optionalString}}",
						"value2": "{{optionalNumber}}",
						"value3": "{{optionalBoolean}}",
						"value4": "{{optionalArray}}",
					},
				},
			},
		},
	}

	// Only provide required parameter
	params := map[string]interface{}{
		"requiredParam": "required-value",
	}

	wf, err := workflow.InstantiateTemplate(context.Background(), template, params)
	if err != nil {
		t.Fatalf("InstantiateTemplate() failed: %v", err)
	}

	// Verify defaults were used
	node := findNodeByID(wf.Nodes, "node1")
	if node == nil {
		t.Fatal("Expected node1 to exist")
	}

	config := node.GetConfiguration()
	if config["value1"] != "default-string" {
		t.Errorf("Expected default string value, got %v", config["value1"])
	}
	if config["value2"] != 42 {
		t.Errorf("Expected default number value 42, got %v", config["value2"])
	}
	if config["value3"] != true {
		t.Errorf("Expected default boolean value true, got %v", config["value3"])
	}
}

// TestTemplateParameterTypeValidation tests that parameter type validation
// catches type mismatches
func TestTemplateParameterTypeValidation(t *testing.T) {
	tests := []struct {
		name      string
		paramType workflow.ParameterType
		value     interface{}
		wantErr   bool
	}{
		{
			name:      "string parameter with string value",
			paramType: workflow.ParameterTypeString,
			value:     "hello",
			wantErr:   false,
		},
		{
			name:      "string parameter with number value",
			paramType: workflow.ParameterTypeString,
			value:     123,
			wantErr:   true,
		},
		{
			name:      "number parameter with integer",
			paramType: workflow.ParameterTypeNumber,
			value:     42,
			wantErr:   false,
		},
		{
			name:      "number parameter with float",
			paramType: workflow.ParameterTypeNumber,
			value:     42.5,
			wantErr:   false,
		},
		{
			name:      "number parameter with string",
			paramType: workflow.ParameterTypeNumber,
			value:     "not a number",
			wantErr:   true,
		},
		{
			name:      "boolean parameter with true",
			paramType: workflow.ParameterTypeBoolean,
			value:     true,
			wantErr:   false,
		},
		{
			name:      "boolean parameter with string",
			paramType: workflow.ParameterTypeBoolean,
			value:     "true",
			wantErr:   true,
		},
		{
			name:      "array parameter with slice",
			paramType: workflow.ParameterTypeArray,
			value:     []interface{}{"a", "b", "c"},
			wantErr:   false,
		},
		{
			name:      "array parameter with string",
			paramType: workflow.ParameterTypeArray,
			value:     "not an array",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template := &workflow.WorkflowTemplate{
				Name:    "type-test-template",
				Version: "1.0.0",
				Parameters: []workflow.TemplateParameter{
					{
						Name:     "testParam",
						Type:     tt.paramType,
						Required: true,
					},
				},
				WorkflowSpec: workflow.WorkflowSpec{
					Nodes: []workflow.NodeSpec{
						{
							ID:   "start",
							Type: "start",
						},
					},
				},
			}

			params := map[string]interface{}{
				"testParam": tt.value,
			}

			_, err := workflow.InstantiateTemplate(context.Background(), template, params)

			if tt.wantErr && err == nil {
				t.Error("Expected type validation error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if tt.wantErr && err != nil && !errors.Is(err, workflow.ErrInvalidParameterType) {
				t.Errorf("Expected ErrInvalidParameterType, got %v", err)
			}
		})
	}
}

// TestTemplateNestedParameterSubstitution tests parameter substitution
// in nested node configurations
func TestTemplateNestedParameterSubstitution(t *testing.T) {
	template := &workflow.WorkflowTemplate{
		Name:    "nested-params-template",
		Version: "1.0.0",
		Parameters: []workflow.TemplateParameter{
			{
				Name:     "serverHost",
				Type:     workflow.ParameterTypeString,
				Required: true,
			},
			{
				Name:     "serverPort",
				Type:     workflow.ParameterTypeNumber,
				Required: true,
			},
			{
				Name:     "apiPath",
				Type:     workflow.ParameterTypeString,
				Required: true,
			},
		},
		WorkflowSpec: workflow.WorkflowSpec{
			Nodes: []workflow.NodeSpec{
				{
					ID:   "api_call",
					Type: "mcp_tool",
					Config: map[string]interface{}{
						"server": "fetch-server",
						"tool":   "fetch",
						"parameters": map[string]interface{}{
							"url": "https://{{serverHost}}:{{serverPort}}/{{apiPath}}",
							"headers": map[string]interface{}{
								"X-Server": "{{serverHost}}",
							},
						},
					},
				},
			},
		},
	}

	params := map[string]interface{}{
		"serverHost": "api.example.com",
		"serverPort": 443,
		"apiPath":    "v1/users",
	}

	wf, err := workflow.InstantiateTemplate(context.Background(), template, params)
	if err != nil {
		t.Fatalf("InstantiateTemplate() failed: %v", err)
	}

	node := findNodeByID(wf.Nodes, "api_call")
	if node == nil {
		t.Fatal("Expected api_call node to exist")
	}

	config := node.GetConfiguration()
	toolParams := config["parameters"].(map[string]interface{})

	expectedURL := "https://api.example.com:443/v1/users"
	if toolParams["url"] != expectedURL {
		t.Errorf("Expected URL %q, got %q", expectedURL, toolParams["url"])
	}

	headers := toolParams["headers"].(map[string]interface{})
	if headers["X-Server"] != "api.example.com" {
		t.Errorf("Expected X-Server header to be 'api.example.com', got %v", headers["X-Server"])
	}
}

// TestTemplateValidationBeforeInstantiation tests that templates are validated
// before instantiation begins
func TestTemplateValidationBeforeInstantiation(t *testing.T) {
	tests := []struct {
		name     string
		template *workflow.WorkflowTemplate
		wantErr  error
	}{
		{
			name: "template with no name",
			template: &workflow.WorkflowTemplate{
				Name:    "",
				Version: "1.0.0",
				WorkflowSpec: workflow.WorkflowSpec{
					Nodes: []workflow.NodeSpec{{ID: "start", Type: "start"}},
				},
			},
			wantErr: workflow.ErrInvalidTemplate,
		},
		{
			name: "template with invalid version",
			template: &workflow.WorkflowTemplate{
				Name:    "test",
				Version: "",
				WorkflowSpec: workflow.WorkflowSpec{
					Nodes: []workflow.NodeSpec{{ID: "start", Type: "start"}},
				},
			},
			wantErr: workflow.ErrInvalidTemplate,
		},
		{
			name: "template with duplicate parameter names",
			template: &workflow.WorkflowTemplate{
				Name:    "test",
				Version: "1.0.0",
				Parameters: []workflow.TemplateParameter{
					{Name: "param1", Type: workflow.ParameterTypeString, Required: true},
					{Name: "param1", Type: workflow.ParameterTypeNumber, Required: true},
				},
				WorkflowSpec: workflow.WorkflowSpec{
					Nodes: []workflow.NodeSpec{{ID: "start", Type: "start"}},
				},
			},
			wantErr: workflow.ErrDuplicateParameterName,
		},
		{
			name: "template with invalid parameter type",
			template: &workflow.WorkflowTemplate{
				Name:    "test",
				Version: "1.0.0",
				Parameters: []workflow.TemplateParameter{
					{Name: "param1", Type: "invalid-type", Required: true},
				},
				WorkflowSpec: workflow.WorkflowSpec{
					Nodes: []workflow.NodeSpec{{ID: "start", Type: "start"}},
				},
			},
			wantErr: workflow.ErrInvalidParameterType,
		},
		{
			name: "template with unreferenced parameter in workflow spec",
			template: &workflow.WorkflowTemplate{
				Name:    "test",
				Version: "1.0.0",
				Parameters: []workflow.TemplateParameter{
					{Name: "param1", Type: workflow.ParameterTypeString, Required: true},
				},
				WorkflowSpec: workflow.WorkflowSpec{
					Nodes: []workflow.NodeSpec{
						{
							ID:   "node1",
							Type: "transform",
							Config: map[string]interface{}{
								"value": "{{undefinedParam}}",
							},
						},
					},
				},
			},
			wantErr: workflow.ErrUndefinedParameter,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := workflow.InstantiateTemplate(context.Background(), tt.template, map[string]interface{}{})

			if err == nil {
				t.Fatal("Expected validation error, got nil")
			}

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

// TestTemplateMultipleInstantiations tests that the same template can be
// instantiated multiple times with different parameters
func TestTemplateMultipleInstantiations(t *testing.T) {
	template := &workflow.WorkflowTemplate{
		Name:        "multi-instance-template",
		Description: "Template for testing multiple instantiations",
		Version:     "1.0.0",
		Parameters: []workflow.TemplateParameter{
			{
				Name:     "instanceName",
				Type:     workflow.ParameterTypeString,
				Required: true,
			},
			{
				Name:     "instanceID",
				Type:     workflow.ParameterTypeNumber,
				Required: true,
			},
		},
		WorkflowSpec: workflow.WorkflowSpec{
			Nodes: []workflow.NodeSpec{
				{
					ID:   "start",
					Type: "start",
				},
				{
					ID:   "process",
					Type: "transform",
					Config: map[string]interface{}{
						"name": "{{instanceName}}",
						"id":   "{{instanceID}}",
					},
				},
			},
		},
	}

	// Create first instance
	params1 := map[string]interface{}{
		"instanceName": "instance-1",
		"instanceID":   1,
	}

	wf1, err := workflow.InstantiateTemplate(context.Background(), template, params1)
	if err != nil {
		t.Fatalf("First instantiation failed: %v", err)
	}

	// Create second instance
	params2 := map[string]interface{}{
		"instanceName": "instance-2",
		"instanceID":   2,
	}

	wf2, err := workflow.InstantiateTemplate(context.Background(), template, params2)
	if err != nil {
		t.Fatalf("Second instantiation failed: %v", err)
	}

	// Verify both workflows are distinct
	if wf1.ID == wf2.ID {
		t.Error("Expected different workflow IDs for separate instantiations")
	}

	// Verify first workflow has correct substitutions
	node1 := findNodeByID(wf1.Nodes, "process")
	config1 := node1.GetConfiguration()
	if config1["name"] != "instance-1" {
		t.Errorf("First workflow: expected name 'instance-1', got %v", config1["name"])
	}
	if config1["id"] != 1 {
		t.Errorf("First workflow: expected id 1, got %v", config1["id"])
	}

	// Verify second workflow has correct substitutions
	node2 := findNodeByID(wf2.Nodes, "process")
	config2 := node2.GetConfiguration()
	if config2["name"] != "instance-2" {
		t.Errorf("Second workflow: expected name 'instance-2', got %v", config2["name"])
	}
	if config2["id"] != 2 {
		t.Errorf("Second workflow: expected id 2, got %v", config2["id"])
	}
}

// TestTemplateWithConditionalSections tests templates with nodes that should
// be included/excluded based on parameter values
func TestTemplateWithConditionalSections(t *testing.T) {
	template := &workflow.WorkflowTemplate{
		Name:    "conditional-template",
		Version: "1.0.0",
		Parameters: []workflow.TemplateParameter{
			{
				Name:     "enableLogging",
				Type:     workflow.ParameterTypeBoolean,
				Required: false,
				Default:  false,
			},
			{
				Name:     "enableRetry",
				Type:     workflow.ParameterTypeBoolean,
				Required: false,
				Default:  false,
			},
		},
		WorkflowSpec: workflow.WorkflowSpec{
			Nodes: []workflow.NodeSpec{
				{
					ID:   "start",
					Type: "start",
				},
				{
					ID:        "log_start",
					Type:      "transform",
					Condition: "{{enableLogging}}",
					Config: map[string]interface{}{
						"message": "Workflow started",
					},
				},
				{
					ID:   "main_process",
					Type: "mcp_tool",
					Config: map[string]interface{}{
						"server": "fetch-server",
						"tool":   "fetch",
					},
				},
				{
					ID:        "retry_handler",
					Type:      "condition",
					Condition: "{{enableRetry}}",
					Config: map[string]interface{}{
						"maxRetries": 3,
					},
				},
				{
					ID:   "end",
					Type: "end",
				},
			},
		},
	}

	t.Run("with logging enabled", func(t *testing.T) {
		params := map[string]interface{}{
			"enableLogging": true,
			"enableRetry":   false,
		}

		wf, err := workflow.InstantiateTemplate(context.Background(), template, params)
		if err != nil {
			t.Fatalf("InstantiateTemplate() failed: %v", err)
		}

		// Verify log_start node is included
		logNode := findNodeByID(wf.Nodes, "log_start")
		if logNode == nil {
			t.Error("Expected log_start node to be included when enableLogging=true")
		}

		// Verify retry_handler node is not included
		retryNode := findNodeByID(wf.Nodes, "retry_handler")
		if retryNode != nil {
			t.Error("Expected retry_handler node to be excluded when enableRetry=false")
		}
	})

	t.Run("with retry enabled", func(t *testing.T) {
		params := map[string]interface{}{
			"enableLogging": false,
			"enableRetry":   true,
		}

		wf, err := workflow.InstantiateTemplate(context.Background(), template, params)
		if err != nil {
			t.Fatalf("InstantiateTemplate() failed: %v", err)
		}

		// Verify log_start node is not included
		logNode := findNodeByID(wf.Nodes, "log_start")
		if logNode != nil {
			t.Error("Expected log_start node to be excluded when enableLogging=false")
		}

		// Verify retry_handler node is included
		retryNode := findNodeByID(wf.Nodes, "retry_handler")
		if retryNode == nil {
			t.Error("Expected retry_handler node to be included when enableRetry=true")
		}
	})
}

// TestTemplateWithParameterSchema tests validation of parameters against
// additional constraints like min/max, regex patterns, etc.
func TestTemplateWithParameterSchema(t *testing.T) {
	template := &workflow.WorkflowTemplate{
		Name:    "schema-template",
		Version: "1.0.0",
		Parameters: []workflow.TemplateParameter{
			{
				Name:     "port",
				Type:     workflow.ParameterTypeNumber,
				Required: true,
				Validation: &workflow.ParameterValidation{
					Min: 1,
					Max: 65535,
				},
			},
			{
				Name:     "email",
				Type:     workflow.ParameterTypeString,
				Required: true,
				Validation: &workflow.ParameterValidation{
					Pattern: `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
				},
			},
			{
				Name:     "tags",
				Type:     workflow.ParameterTypeArray,
				Required: false,
				Validation: &workflow.ParameterValidation{
					MinLength: 1,
					MaxLength: 5,
				},
			},
		},
		WorkflowSpec: workflow.WorkflowSpec{
			Nodes: []workflow.NodeSpec{{ID: "start", Type: "start"}},
		},
	}

	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid parameters",
			params: map[string]interface{}{
				"port":  8080,
				"email": "user@example.com",
				"tags":  []interface{}{"tag1", "tag2"},
			},
			wantErr: false,
		},
		{
			name: "port below minimum",
			params: map[string]interface{}{
				"port":  0,
				"email": "user@example.com",
			},
			wantErr: true,
		},
		{
			name: "port above maximum",
			params: map[string]interface{}{
				"port":  70000,
				"email": "user@example.com",
			},
			wantErr: true,
		},
		{
			name: "invalid email format",
			params: map[string]interface{}{
				"port":  8080,
				"email": "not-an-email",
			},
			wantErr: true,
		},
		{
			name: "array too long",
			params: map[string]interface{}{
				"port":  8080,
				"email": "user@example.com",
				"tags":  []interface{}{"t1", "t2", "t3", "t4", "t5", "t6"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := workflow.InstantiateTemplate(context.Background(), template, tt.params)

			if tt.wantErr && err == nil {
				t.Error("Expected validation error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

// Helper function to find a node by ID
func findNodeByID(nodes []workflow.Node, id string) workflow.Node {
	for _, node := range nodes {
		if node.GetID() == id {
			return node
		}
	}
	return nil
}
