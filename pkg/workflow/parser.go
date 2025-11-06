package workflow

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// yamlWorkflow represents the YAML structure before conversion to domain objects
type yamlWorkflow struct {
	Version     string             `yaml:"version"`
	Name        string             `yaml:"name"`
	Description string             `yaml:"description,omitempty"`
	Metadata    *WorkflowMetadata  `yaml:"metadata,omitempty"`
	Variables   []yamlVariable     `yaml:"variables,omitempty"`
	Servers     []yamlServerConfig `yaml:"servers,omitempty"`
	Nodes       []yamlNode         `yaml:"nodes,omitempty"`
	Edges       []yamlEdge         `yaml:"edges,omitempty"`
}

// yamlVariable represents a variable in YAML before type conversion
type yamlVariable struct {
	Name         string      `yaml:"name"`
	Type         string      `yaml:"type"`
	DefaultValue interface{} `yaml:"default,omitempty"`
	Description  string      `yaml:"description,omitempty"`
}

// yamlServerConfig represents a server config in YAML
type yamlServerConfig struct {
	ID            string            `yaml:"id"`
	Name          string            `yaml:"name,omitempty"`
	Command       string            `yaml:"command"`
	Args          []string          `yaml:"args,omitempty"`
	Transport     string            `yaml:"transport,omitempty"`
	Env           map[string]string `yaml:"env,omitempty"`
	CredentialRef string            `yaml:"credential_ref,omitempty"`
}

// yamlNode represents a node in YAML with type-specific fields
type yamlNode struct {
	ID   string `yaml:"id"`
	Type string `yaml:"type"`

	// EndNode fields
	Return string `yaml:"return,omitempty"`

	// MCPToolNode fields
	Server     string            `yaml:"server,omitempty"`
	Tool       string            `yaml:"tool,omitempty"`
	Parameters map[string]string `yaml:"parameters,omitempty"`
	Output     string            `yaml:"output,omitempty"`

	// TransformNode fields
	Input      string `yaml:"input,omitempty"`
	Expression string `yaml:"expression,omitempty"`

	// ConditionNode fields
	Condition string `yaml:"condition,omitempty"`

	// ParallelNode fields
	Branches [][]string `yaml:"branches,omitempty"`
	Merge    string     `yaml:"merge,omitempty"`

	// LoopNode fields
	Collection     string   `yaml:"collection,omitempty"`
	Item           string   `yaml:"item,omitempty"`
	Body           []string `yaml:"body,omitempty"`
	BreakCondition string   `yaml:"break_condition,omitempty"`
}

// yamlEdge represents an edge in YAML
type yamlEdge struct {
	From      string `yaml:"from"`
	To        string `yaml:"to"`
	Condition string `yaml:"condition,omitempty"`
	Label     string `yaml:"label,omitempty"`
}

// Parse parses a workflow from YAML bytes
func Parse(yamlBytes []byte) (*Workflow, error) {
	if len(yamlBytes) == 0 {
		return nil, errors.New("empty YAML input")
	}

	var yw yamlWorkflow
	if err := yaml.Unmarshal(yamlBytes, &yw); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate required fields
	if yw.Version == "" {
		return nil, errors.New("missing required field: version")
	}
	if yw.Name == "" {
		return nil, errors.New("missing required field: name")
	}

	// Create workflow
	wf := &Workflow{
		ID:            NewWorkflowID().String(),
		Name:          yw.Name,
		Version:       yw.Version,
		Description:   yw.Description,
		Variables:     make([]*Variable, 0),
		ServerConfigs: make([]*ServerConfig, 0),
		Nodes:         make([]Node, 0),
		Edges:         make([]*Edge, 0),
	}

	// Set metadata (use defaults if not provided)
	if yw.Metadata != nil {
		wf.Metadata = *yw.Metadata
	}

	// Parse variables
	for _, yv := range yw.Variables {
		variable := &Variable{
			Name:         yv.Name,
			Type:         yv.Type,
			DefaultValue: yv.DefaultValue,
			Description:  yv.Description,
		}
		if err := wf.AddVariable(variable); err != nil {
			return nil, fmt.Errorf("failed to add variable: %w", err)
		}
	}

	// Parse server configs
	for _, ys := range yw.Servers {
		serverConfig := &ServerConfig{
			ID:            ys.ID,
			Name:          ys.Name,
			Command:       ys.Command,
			Args:          ys.Args,
			Transport:     ys.Transport,
			Env:           ys.Env,
			CredentialRef: ys.CredentialRef,
		}
		// Validate server config
		if err := serverConfig.Validate(); err != nil {
			return nil, fmt.Errorf("invalid server config: %w", err)
		}
		wf.ServerConfigs = append(wf.ServerConfigs, serverConfig)
	}

	// Parse nodes
	for _, yn := range yw.Nodes {
		node, err := parseNode(yn)
		if err != nil {
			return nil, fmt.Errorf("failed to parse node '%s': %w", yn.ID, err)
		}
		if err := wf.AddNode(node); err != nil {
			return nil, fmt.Errorf("failed to add node: %w", err)
		}
	}

	// Parse edges
	for _, ye := range yw.Edges {
		edge := &Edge{
			ID:         NewEdgeID().String(),
			FromNodeID: ye.From,
			ToNodeID:   ye.To,
			Condition:  ye.Condition,
			Label:      ye.Label,
		}
		if err := wf.AddEdge(edge); err != nil {
			return nil, fmt.Errorf("failed to add edge: %w", err)
		}
	}

	// Validate the parsed workflow
	if err := wf.Validate(); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	return wf, nil
}

// ParseFile parses a workflow from a YAML file
func ParseFile(filePath string) (*Workflow, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return Parse(data)
}

// parseNode converts a yamlNode to the appropriate concrete Node type
func parseNode(yn yamlNode) (Node, error) {
	if yn.ID == "" {
		return nil, errors.New("node ID cannot be empty")
	}
	if yn.Type == "" {
		return nil, fmt.Errorf("node '%s': type cannot be empty", yn.ID)
	}

	switch yn.Type {
	case "start":
		return &StartNode{ID: yn.ID}, nil

	case "end":
		return &EndNode{
			ID:          yn.ID,
			ReturnValue: yn.Return,
		}, nil

	case "mcp_tool":
		if yn.Server == "" {
			return nil, fmt.Errorf("mcp_tool node '%s': server field is required", yn.ID)
		}
		if yn.Tool == "" {
			return nil, fmt.Errorf("mcp_tool node '%s': tool field is required", yn.ID)
		}
		if yn.Output == "" {
			return nil, fmt.Errorf("mcp_tool node '%s': output field is required", yn.ID)
		}
		return &MCPToolNode{
			ID:             yn.ID,
			ServerID:       yn.Server,
			ToolName:       yn.Tool,
			Parameters:     yn.Parameters,
			OutputVariable: yn.Output,
		}, nil

	case "transform":
		if yn.Input == "" {
			return nil, fmt.Errorf("transform node '%s': input field is required", yn.ID)
		}
		if yn.Expression == "" {
			return nil, fmt.Errorf("transform node '%s': expression field is required", yn.ID)
		}
		if yn.Output == "" {
			return nil, fmt.Errorf("transform node '%s': output field is required", yn.ID)
		}
		return &TransformNode{
			ID:             yn.ID,
			InputVariable:  yn.Input,
			Expression:     yn.Expression,
			OutputVariable: yn.Output,
		}, nil

	case "condition":
		if yn.Condition == "" {
			return nil, fmt.Errorf("condition node '%s': condition field is required", yn.ID)
		}
		return &ConditionNode{
			ID:        yn.ID,
			Condition: yn.Condition,
		}, nil

	case "passthrough":
		return &PassthroughNode{
			ID: yn.ID,
		}, nil

	case "parallel":
		if len(yn.Branches) == 0 {
			return nil, fmt.Errorf("parallel node '%s': branches field is required", yn.ID)
		}
		mergeStrategy := yn.Merge
		if mergeStrategy == "" {
			mergeStrategy = "wait_all" // default
		}
		return &ParallelNode{
			ID:            yn.ID,
			Branches:      yn.Branches,
			MergeStrategy: mergeStrategy,
		}, nil

	case "loop":
		if yn.Collection == "" {
			return nil, fmt.Errorf("loop node '%s': collection field is required", yn.ID)
		}
		if yn.Item == "" {
			return nil, fmt.Errorf("loop node '%s': item field is required", yn.ID)
		}
		if len(yn.Body) == 0 {
			return nil, fmt.Errorf("loop node '%s': body field is required", yn.ID)
		}
		return &LoopNode{
			ID:             yn.ID,
			Collection:     yn.Collection,
			ItemVariable:   yn.Item,
			Body:           yn.Body,
			BreakCondition: yn.BreakCondition,
		}, nil

	default:
		return nil, fmt.Errorf("unknown node type: %s", yn.Type)
	}
}

// ToYAML serializes a workflow to YAML bytes
func ToYAML(workflow *Workflow) ([]byte, error) {
	if workflow == nil {
		return nil, errors.New("workflow cannot be nil")
	}

	// Convert workflow to YAML structure
	yw := yamlWorkflow{
		Version:     workflow.Version,
		Name:        workflow.Name,
		Description: workflow.Description,
		Metadata:    &workflow.Metadata,
		Variables:   make([]yamlVariable, 0, len(workflow.Variables)),
		Servers:     make([]yamlServerConfig, 0, len(workflow.ServerConfigs)),
		Nodes:       make([]yamlNode, 0, len(workflow.Nodes)),
		Edges:       make([]yamlEdge, 0, len(workflow.Edges)),
	}

	// Convert variables
	for _, v := range workflow.Variables {
		yw.Variables = append(yw.Variables, yamlVariable{
			Name:         v.Name,
			Type:         v.Type,
			DefaultValue: v.DefaultValue,
			Description:  v.Description,
		})
	}

	// Convert server configs
	for _, s := range workflow.ServerConfigs {
		yw.Servers = append(yw.Servers, yamlServerConfig{
			ID:            s.ID,
			Name:          s.Name,
			Command:       s.Command,
			Args:          s.Args,
			Transport:     s.Transport,
			Env:           s.Env,
			CredentialRef: s.CredentialRef,
		})
	}

	// Convert nodes
	for _, node := range workflow.Nodes {
		yn, err := nodeToYAML(node)
		if err != nil {
			return nil, fmt.Errorf("failed to convert node to YAML: %w", err)
		}
		yw.Nodes = append(yw.Nodes, yn)
	}

	// Convert edges
	for _, edge := range workflow.Edges {
		yw.Edges = append(yw.Edges, yamlEdge{
			From:      edge.FromNodeID,
			To:        edge.ToNodeID,
			Condition: edge.Condition,
			Label:     edge.Label,
		})
	}

	// Marshal to YAML
	yamlBytes, err := yaml.Marshal(&yw)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal to YAML: %w", err)
	}

	return yamlBytes, nil
}

// nodeToYAML converts a Node interface to yamlNode
func nodeToYAML(node Node) (yamlNode, error) {
	yn := yamlNode{
		ID:   node.GetID(),
		Type: node.Type(),
	}

	switch n := node.(type) {
	case *StartNode:
		// No additional fields

	case *EndNode:
		yn.Return = n.ReturnValue

	case *MCPToolNode:
		yn.Server = n.ServerID
		yn.Tool = n.ToolName
		yn.Parameters = n.Parameters
		yn.Output = n.OutputVariable

	case *TransformNode:
		yn.Input = n.InputVariable
		yn.Expression = n.Expression
		yn.Output = n.OutputVariable

	case *ConditionNode:
		yn.Condition = n.Condition

	case *ParallelNode:
		yn.Branches = n.Branches
		yn.Merge = n.MergeStrategy

	case *LoopNode:
		yn.Collection = n.Collection
		yn.Item = n.ItemVariable
		yn.Body = n.Body
		yn.BreakCondition = n.BreakCondition

	default:
		return yn, fmt.Errorf("unknown node type: %T", node)
	}

	return yn, nil
}
