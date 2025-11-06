package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type WorkflowTemplate struct {
	Name         string      `yaml:"name"`
	Description  string      `yaml:"description"`
	Version      string      `yaml:"version"`
	Parameters   []Parameter `yaml:"parameters"`
	WorkflowSpec struct {
		Version string        `yaml:"version"`
		Name    string        `yaml:"name"`
		Servers []interface{} `yaml:"servers"`
		Nodes   []interface{} `yaml:"nodes"`
		Edges   []interface{} `yaml:"edges"`
	} `yaml:"workflow_spec"`
}

type Parameter struct {
	Name        string                 `yaml:"name"`
	Type        string                 `yaml:"type"`
	Required    bool                   `yaml:"required"`
	Default     interface{}            `yaml:"default,omitempty"`
	Description string                 `yaml:"description,omitempty"`
	Validation  map[string]interface{} `yaml:"validation,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <template-file>\n", os.Args[0])
		os.Exit(1)
	}

	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	var template WorkflowTemplate
	if err := yaml.Unmarshal(data, &template); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing YAML: %v\n", err)
		os.Exit(1)
	}

	// Basic validation
	if template.Name == "" {
		fmt.Fprintf(os.Stderr, "Error: template name is required\n")
		os.Exit(1)
	}

	if template.Version == "" {
		fmt.Fprintf(os.Stderr, "Error: template version is required\n")
		os.Exit(1)
	}

	if template.WorkflowSpec.Version == "" {
		fmt.Fprintf(os.Stderr, "Error: workflow_spec.version is required\n")
		os.Exit(1)
	}

	if len(template.WorkflowSpec.Nodes) == 0 {
		fmt.Fprintf(os.Stderr, "Error: workflow_spec.nodes cannot be empty\n")
		os.Exit(1)
	}

	fmt.Printf("✓ Template '%s' v%s is valid\n", template.Name, template.Version)
	fmt.Printf("  - Parameters: %d\n", len(template.Parameters))
	fmt.Printf("  - Servers: %d\n", len(template.WorkflowSpec.Servers))
	fmt.Printf("  - Nodes: %d\n", len(template.WorkflowSpec.Nodes))
	fmt.Printf("  - Edges: %d\n", len(template.WorkflowSpec.Edges))
}
