package cli

import (
	"fmt"
	"os"

	"github.com/dshills/goflow/pkg/workflow"
	"gopkg.in/yaml.v3"
)

// WorkflowYAML is a temporary structure for loading YAML before converting to Workflow
type WorkflowYAML struct {
	Version       string                    `yaml:"version"`
	Name          string                    `yaml:"name"`
	Description   string                    `yaml:"description,omitempty"`
	Metadata      workflow.WorkflowMetadata `yaml:"metadata,omitempty"`
	Variables     []*workflow.Variable      `yaml:"variables,omitempty"`
	ServerConfigs []*workflow.ServerConfig  `yaml:"servers,omitempty"`
	Nodes         []map[string]interface{}  `yaml:"nodes,omitempty"`
	Edges         []*workflow.Edge          `yaml:"edges,omitempty"`
}

// LoadWorkflowFromFile loads a workflow from a YAML file
func LoadWorkflowFromFile(path string) (*workflow.Workflow, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file: %w", err)
	}

	// Parse YAML into intermediate structure
	var yamlWf WorkflowYAML
	if err := yaml.Unmarshal(data, &yamlWf); err != nil {
		return nil, fmt.Errorf("failed to parse workflow YAML: %w", err)
	}

	// Create workflow
	wf := &workflow.Workflow{
		ID:            yamlWf.Name, // Use name as ID for now
		Name:          yamlWf.Name,
		Version:       yamlWf.Version,
		Description:   yamlWf.Description,
		Metadata:      yamlWf.Metadata,
		Variables:     yamlWf.Variables,
		ServerConfigs: yamlWf.ServerConfigs,
		Nodes:         make([]workflow.Node, 0),
		Edges:         make([]*workflow.Edge, 0),
	}

	// Convert node maps to concrete node types
	for _, nodeMap := range yamlWf.Nodes {
		node, err := nodeMapToNode(nodeMap)
		if err != nil {
			return nil, fmt.Errorf("failed to convert node: %w", err)
		}
		wf.Nodes = append(wf.Nodes, node)
	}

	// Process edges to add IDs if missing
	for _, edge := range yamlWf.Edges {
		if edge == nil {
			continue
		}
		// Generate ID if not present
		if edge.ID == "" {
			edge.ID = fmt.Sprintf("edge-%s-%s", edge.FromNodeID, edge.ToNodeID)
		}
		wf.Edges = append(wf.Edges, edge)
	}

	return wf, nil
}

// nodeMapToNode converts a map to a concrete Node type
func nodeMapToNode(nodeMap map[string]interface{}) (workflow.Node, error) {
	nodeType, ok := nodeMap["type"].(string)
	if !ok {
		return nil, fmt.Errorf("node missing 'type' field")
	}

	id, ok := nodeMap["id"].(string)
	if !ok {
		return nil, fmt.Errorf("node missing 'id' field")
	}

	switch nodeType {
	case "start":
		return &workflow.StartNode{
			ID: id,
		}, nil

	case "end":
		node := &workflow.EndNode{
			ID: id,
		}
		if returnValue, ok := nodeMap["return_value"].(string); ok {
			node.ReturnValue = returnValue
		} else if returnValue, ok := nodeMap["return"].(string); ok {
			node.ReturnValue = returnValue
		}
		return node, nil

	case "mcp_tool":
		node := &workflow.MCPToolNode{
			ID: id,
		}
		if server, ok := nodeMap["server"].(string); ok {
			node.ServerID = server
		}
		if tool, ok := nodeMap["tool"].(string); ok {
			node.ToolName = tool
		}
		if params, ok := nodeMap["parameters"].(map[string]interface{}); ok {
			// Convert map[string]interface{} to map[string]string
			node.Parameters = make(map[string]string)
			for k, v := range params {
				node.Parameters[k] = fmt.Sprintf("%v", v)
			}
		}
		if output, ok := nodeMap["output"].(string); ok {
			node.OutputVariable = output
		}
		return node, nil

	case "transform":
		node := &workflow.TransformNode{
			ID: id,
		}
		if input, ok := nodeMap["input"].(string); ok {
			node.InputVariable = input
		}
		if expr, ok := nodeMap["expression"].(string); ok {
			node.Expression = expr
		}
		if output, ok := nodeMap["output"].(string); ok {
			node.OutputVariable = output
		}
		return node, nil

	case "condition":
		node := &workflow.ConditionNode{
			ID: id,
		}
		if condition, ok := nodeMap["condition"].(string); ok {
			node.Condition = condition
		}
		return node, nil

	default:
		return nil, fmt.Errorf("unknown node type: %s", nodeType)
	}
}
