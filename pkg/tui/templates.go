package tui

import (
	"github.com/dshills/goflow/pkg/workflow"
)

// WorkflowTemplates maps template names to template creation functions
var WorkflowTemplates = map[string]func() *workflow.Workflow{
	"basic":           CreateBasicTemplate,
	"etl":             CreateETLTemplate,
	"api-integration": CreateAPIIntegrationTemplate,
}

// TemplateDescriptions provides user-friendly descriptions for each template
var TemplateDescriptions = map[string]string{
	"basic":           "Simple workflow with 3 nodes",
	"etl":             "Extract, Transform, Load pipeline",
	"api-integration": "API call with error handling and retry",
}

// CreateBasicTemplate creates a simple 3-node workflow: Start → MCP Tool → End
func CreateBasicTemplate() *workflow.Workflow {
	wf, _ := workflow.NewWorkflow("basic-workflow", "Basic workflow template")

	// Add a placeholder server config for the template
	wf.ServerConfigs = []*workflow.ServerConfig{
		{
			ID:      "my-server",
			Command: "example-mcp-server",
		},
	}

	// Create nodes
	start := &workflow.StartNode{ID: "start"}
	tool := &workflow.MCPToolNode{
		ID:             "mcp-tool-1",
		ServerID:       "my-server",
		ToolName:       "example-tool",
		OutputVariable: "result",
		Parameters:     make(map[string]string),
	}
	end := &workflow.EndNode{
		ID:          "end",
		ReturnValue: "${result}",
	}

	// Add nodes to workflow
	wf.AddNode(start)
	wf.AddNode(tool)
	wf.AddNode(end)

	// Add edges
	wf.AddEdge(&workflow.Edge{
		FromNodeID: "start",
		ToNodeID:   "mcp-tool-1",
	})
	wf.AddEdge(&workflow.Edge{
		FromNodeID: "mcp-tool-1",
		ToNodeID:   "end",
	})

	return wf
}

// CreateETLTemplate creates an Extract, Transform, Load pipeline workflow
func CreateETLTemplate() *workflow.Workflow {
	wf, _ := workflow.NewWorkflow("etl-workflow", "ETL pipeline template")

	// Add a placeholder server config for the template
	wf.ServerConfigs = []*workflow.ServerConfig{
		{
			ID:      "data-server",
			Command: "data-mcp-server",
		},
	}

	// Create nodes
	start := &workflow.StartNode{ID: "start"}
	extract := &workflow.MCPToolNode{
		ID:             "extract",
		ServerID:       "data-server",
		ToolName:       "extract-tool",
		OutputVariable: "raw_data",
		Parameters:     make(map[string]string),
	}
	transform := &workflow.TransformNode{
		ID:             "transform",
		InputVariable:  "raw_data",
		Expression:     "$.data",
		OutputVariable: "cleaned_data",
	}
	load := &workflow.MCPToolNode{
		ID:             "load",
		ServerID:       "data-server",
		ToolName:       "load-tool",
		OutputVariable: "load_result",
		Parameters:     make(map[string]string),
	}
	end := &workflow.EndNode{
		ID:          "end",
		ReturnValue: "${load_result}",
	}

	// Add nodes to workflow
	wf.AddNode(start)
	wf.AddNode(extract)
	wf.AddNode(transform)
	wf.AddNode(load)
	wf.AddNode(end)

	// Add edges
	wf.AddEdge(&workflow.Edge{
		FromNodeID: "start",
		ToNodeID:   "extract",
	})
	wf.AddEdge(&workflow.Edge{
		FromNodeID: "extract",
		ToNodeID:   "transform",
	})
	wf.AddEdge(&workflow.Edge{
		FromNodeID: "transform",
		ToNodeID:   "load",
	})
	wf.AddEdge(&workflow.Edge{
		FromNodeID: "load",
		ToNodeID:   "end",
	})

	return wf
}

// CreateAPIIntegrationTemplate creates an API integration workflow with error handling and retry
func CreateAPIIntegrationTemplate() *workflow.Workflow {
	wf, _ := workflow.NewWorkflow("api-integration-workflow", "API integration with error handling")

	// Add a placeholder server config for the template
	wf.ServerConfigs = []*workflow.ServerConfig{
		{
			ID:      "api-server",
			Command: "api-mcp-server",
		},
	}

	// Add a workflow variable for the API response
	wf.Variables = []*workflow.Variable{
		{
			Name: "api_response",
			Type: "object",
		},
	}

	// Create nodes
	start := &workflow.StartNode{ID: "start"}
	apiCall := &workflow.MCPToolNode{
		ID:             "api-call",
		ServerID:       "api-server",
		ToolName:       "http-request",
		OutputVariable: "api_response",
		Parameters:     make(map[string]string),
	}
	// Use a simple condition expression that will validate
	checkStatus := &workflow.ConditionNode{
		ID:        "check-status",
		Condition: "api_response != nil",
	}
	retryLoop := &workflow.LoopNode{
		ID:           "retry-loop",
		Collection:   "[1, 2, 3]",
		ItemVariable: "attempt",
		Body:         []string{"api-call"},
	}
	success := &workflow.EndNode{
		ID:          "success",
		ReturnValue: "${api_response}",
	}
	failure := &workflow.EndNode{
		ID:          "failure",
		ReturnValue: "error",
	}

	// Add nodes to workflow
	wf.AddNode(start)
	wf.AddNode(apiCall)
	wf.AddNode(checkStatus)
	wf.AddNode(retryLoop)
	wf.AddNode(success)
	wf.AddNode(failure)

	// Add edges
	wf.AddEdge(&workflow.Edge{
		FromNodeID: "start",
		ToNodeID:   "api-call",
	})
	wf.AddEdge(&workflow.Edge{
		FromNodeID: "api-call",
		ToNodeID:   "check-status",
	})
	wf.AddEdge(&workflow.Edge{
		FromNodeID: "check-status",
		ToNodeID:   "success",
		Condition:  "true",
	})
	wf.AddEdge(&workflow.Edge{
		FromNodeID: "check-status",
		ToNodeID:   "retry-loop",
		Condition:  "false",
	})
	wf.AddEdge(&workflow.Edge{
		FromNodeID: "retry-loop",
		ToNodeID:   "failure",
	})

	return wf
}
