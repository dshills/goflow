package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/dshills/goflow/pkg/tui"
	"github.com/dshills/goflow/pkg/workflow"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// WorkflowTemplate defines different workflow templates
type WorkflowTemplate string

const (
	TemplateBasic          WorkflowTemplate = "basic"
	TemplateETL            WorkflowTemplate = "etl"
	TemplateAPIIntegration WorkflowTemplate = "api-integration"
	TemplateBatchProcess   WorkflowTemplate = "batch-processing"
)

// NewInitCommand creates the init command
func NewInitCommand() *cobra.Command {
	var (
		description string
		template    string
		edit        bool
	)

	cmd := &cobra.Command{
		Use:   "init <workflow-name>",
		Short: "Initialize a new workflow",
		Long: `Create a new workflow file with a basic template.

The workflow is created in ~/.goflow/workflows/<workflow-name>.yaml

Examples:
  goflow init my-workflow
  goflow init data-pipeline --description "ETL pipeline for customer data"
  goflow init api-workflow --template api-integration
  goflow init my-workflow --edit  # Create and open in TUI editor`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workflowName := args[0]

			// Validate workflow name
			if !isValidWorkflowName(workflowName) {
				return fmt.Errorf("invalid workflow name: %s\n\nWorkflow names must:\n  - Start with a letter\n  - Contain only letters, numbers, hyphens, and underscores\n  - Be between 1 and 64 characters", workflowName)
			}

			// Construct workflow path
			workflowPath := filepath.Join(GetWorkflowsDir(), workflowName+".yaml")

			// Check if workflow already exists
			if _, err := os.Stat(workflowPath); err == nil {
				return fmt.Errorf("workflow already exists: %s\n\nLocation: %s", workflowName, workflowPath)
			}

			// Create workflow using the specified template
			var wf *workflow.Workflow
			var err error

			if template != "" {
				wf, err = createWorkflowFromTemplate(workflowName, description, WorkflowTemplate(template))
				if err != nil {
					return fmt.Errorf("failed to create workflow from template: %w", err)
				}
			} else {
				// Create default workflow template
				wf, err = createBasicWorkflow(workflowName, description)
				if err != nil {
					return fmt.Errorf("failed to create workflow: %w", err)
				}
			}

			// Convert workflow to YAML-friendly format
			workflowYAML := workflowToYAMLMap(wf)

			// Marshal to YAML
			data, err := yaml.Marshal(workflowYAML)
			if err != nil {
				return fmt.Errorf("failed to marshal workflow: %w", err)
			}

			// Write to file
			if err := os.WriteFile(workflowPath, data, 0644); err != nil {
				return fmt.Errorf("failed to write workflow file: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Created workflow: %s\n", workflowName)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Location: %s\n", workflowPath)

			// Launch TUI editor if --edit flag is set
			if edit {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nLaunching TUI editor...")
				return launchTUIForWorkflow(workflowName)
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nNext steps:")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  1. Edit the workflow: goflow edit %s\n", workflowName)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  2. Validate: goflow validate %s\n", workflowName)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  3. Execute: goflow run %s\n", workflowName)

			return nil
		},
	}

	cmd.Flags().StringVarP(&description, "description", "d", "", "Workflow description")
	cmd.Flags().StringVarP(&template, "template", "t", "", "Template to use (basic, etl, api-integration, batch-processing)")
	cmd.Flags().BoolVar(&edit, "edit", false, "Open workflow in TUI editor after creation")

	return cmd
}

// isValidWorkflowName validates workflow name format
func isValidWorkflowName(name string) bool {
	// Must start with letter, contain only alphanumeric, hyphens, underscores
	// Length between 1 and 64 characters
	pattern := `^[a-zA-Z][a-zA-Z0-9_-]{0,63}$`
	matched, _ := regexp.MatchString(pattern, name)
	return matched
}

// createBasicWorkflow creates a basic workflow with start and end nodes
func createBasicWorkflow(name, description string) (*workflow.Workflow, error) {
	wf, err := workflow.NewWorkflow(name, description)
	if err != nil {
		return nil, err
	}

	// Add start node
	startNode := &workflow.StartNode{ID: "start"}
	if err := wf.AddNode(startNode); err != nil {
		return nil, fmt.Errorf("failed to add start node: %w", err)
	}

	// Add end node
	endNode := &workflow.EndNode{ID: "end"}
	if err := wf.AddNode(endNode); err != nil {
		return nil, fmt.Errorf("failed to add end node: %w", err)
	}

	// Connect start to end
	edge := &workflow.Edge{
		FromNodeID: "start",
		ToNodeID:   "end",
	}
	if err := wf.AddEdge(edge); err != nil {
		return nil, fmt.Errorf("failed to add edge: %w", err)
	}

	return wf, nil
}

// createWorkflowFromTemplate creates a workflow from a predefined template
func createWorkflowFromTemplate(name, description string, tmpl WorkflowTemplate) (*workflow.Workflow, error) {
	switch tmpl {
	case TemplateETL:
		return createETLTemplate(name, description)
	case TemplateAPIIntegration:
		return createAPIIntegrationTemplate(name, description)
	case TemplateBatchProcess:
		return createBatchProcessingTemplate(name, description)
	case TemplateBasic:
		return createBasicWorkflow(name, description)
	default:
		return nil, fmt.Errorf("unknown template: %s", tmpl)
	}
}

// createETLTemplate creates an ETL workflow template
func createETLTemplate(name, description string) (*workflow.Workflow, error) {
	if description == "" {
		description = "ETL pipeline workflow"
	}

	wf, err := workflow.NewWorkflow(name, description)
	if err != nil {
		return nil, err
	}

	// Add nodes
	startNode := &workflow.StartNode{ID: "start"}
	extractNode := &workflow.MCPToolNode{ID: "extract", ServerID: "data-server", ToolName: "extract_data"}
	transformNode := &workflow.TransformNode{ID: "transform", InputVariable: "raw_data", OutputVariable: "processed_data"}
	loadNode := &workflow.MCPToolNode{ID: "load", ServerID: "data-server", ToolName: "load_data"}
	endNode := &workflow.EndNode{ID: "end"}

	wf.AddNode(startNode)
	wf.AddNode(extractNode)
	wf.AddNode(transformNode)
	wf.AddNode(loadNode)
	wf.AddNode(endNode)

	// Add edges
	wf.AddEdge(&workflow.Edge{FromNodeID: "start", ToNodeID: "extract"})
	wf.AddEdge(&workflow.Edge{FromNodeID: "extract", ToNodeID: "transform"})
	wf.AddEdge(&workflow.Edge{FromNodeID: "transform", ToNodeID: "load"})
	wf.AddEdge(&workflow.Edge{FromNodeID: "load", ToNodeID: "end"})

	return wf, nil
}

// createAPIIntegrationTemplate creates an API integration workflow template
func createAPIIntegrationTemplate(name, description string) (*workflow.Workflow, error) {
	if description == "" {
		description = "API integration workflow"
	}

	wf, err := workflow.NewWorkflow(name, description)
	if err != nil {
		return nil, err
	}

	// Add nodes
	startNode := &workflow.StartNode{ID: "start"}
	fetchNode := &workflow.MCPToolNode{ID: "fetch_api", ServerID: "http-server", ToolName: "http_get"}
	processNode := &workflow.TransformNode{ID: "process_response", InputVariable: "api_response", OutputVariable: "result"}
	endNode := &workflow.EndNode{ID: "end"}

	wf.AddNode(startNode)
	wf.AddNode(fetchNode)
	wf.AddNode(processNode)
	wf.AddNode(endNode)

	// Add edges
	wf.AddEdge(&workflow.Edge{FromNodeID: "start", ToNodeID: "fetch_api"})
	wf.AddEdge(&workflow.Edge{FromNodeID: "fetch_api", ToNodeID: "process_response"})
	wf.AddEdge(&workflow.Edge{FromNodeID: "process_response", ToNodeID: "end"})

	return wf, nil
}

// createBatchProcessingTemplate creates a batch processing workflow template
func createBatchProcessingTemplate(name, description string) (*workflow.Workflow, error) {
	if description == "" {
		description = "Batch processing workflow"
	}

	wf, err := workflow.NewWorkflow(name, description)
	if err != nil {
		return nil, err
	}

	// Add nodes
	startNode := &workflow.StartNode{ID: "start"}
	processNode := &workflow.MCPToolNode{ID: "process_batch", ServerID: "batch-server", ToolName: "process_items"}
	endNode := &workflow.EndNode{ID: "end"}

	wf.AddNode(startNode)
	wf.AddNode(processNode)
	wf.AddNode(endNode)

	// Add edges
	wf.AddEdge(&workflow.Edge{FromNodeID: "start", ToNodeID: "process_batch"})
	wf.AddEdge(&workflow.Edge{FromNodeID: "process_batch", ToNodeID: "end"})

	return wf, nil
}

// workflowToYAMLMap converts a Workflow to a map suitable for YAML marshaling
func workflowToYAMLMap(wf *workflow.Workflow) map[string]interface{} {
	nodes := make([]map[string]interface{}, 0, len(wf.Nodes))
	for _, node := range wf.Nodes {
		nodeMap := nodeToMap(node)
		nodes = append(nodes, nodeMap)
	}

	edges := make([]map[string]interface{}, 0, len(wf.Edges))
	for _, edge := range wf.Edges {
		edges = append(edges, map[string]interface{}{
			"from": edge.FromNodeID,
			"to":   edge.ToNodeID,
		})
	}

	return map[string]interface{}{
		"version":     wf.Version,
		"name":        wf.Name,
		"description": wf.Description,
		"metadata": map[string]interface{}{
			"author":  wf.Metadata.Author,
			"created": wf.Metadata.Created.Format(time.RFC3339),
			"tags":    wf.Metadata.Tags,
		},
		"variables": wf.Variables,
		"servers":   wf.ServerConfigs,
		"nodes":     nodes,
		"edges":     edges,
	}
}

// nodeToMap converts a Node to a map for YAML marshaling
func nodeToMap(node workflow.Node) map[string]interface{} {
	m := map[string]interface{}{
		"id":   node.GetID(),
		"type": node.Type(),
	}

	switch n := node.(type) {
	case *workflow.MCPToolNode:
		if n.ServerID != "" {
			m["server"] = n.ServerID
		}
		if n.ToolName != "" {
			m["tool"] = n.ToolName
		}
		if len(n.Parameters) > 0 {
			m["parameters"] = n.Parameters
		}
		if n.OutputVariable != "" {
			m["output"] = n.OutputVariable
		}
	case *workflow.TransformNode:
		if n.InputVariable != "" {
			m["input"] = n.InputVariable
		}
		if n.Expression != "" {
			m["expression"] = n.Expression
		}
		if n.OutputVariable != "" {
			m["output"] = n.OutputVariable
		}
	case *workflow.ConditionNode:
		if n.Condition != "" {
			m["condition"] = n.Condition
		}
	case *workflow.EndNode:
		if n.ReturnValue != "" {
			m["return"] = n.ReturnValue
		}
	}

	return m
}

// launchTUIForWorkflow launches the TUI editor for a specific workflow
func launchTUIForWorkflow(workflowName string) error {
	app, err := tui.NewApp()
	if err != nil {
		return fmt.Errorf("failed to initialize TUI: %w", err)
	}
	defer app.Close()

	// Get the builder view and set the workflow
	view, err := app.GetViewManager().GetView("builder")
	if err != nil {
		return fmt.Errorf("failed to get builder view: %w", err)
	}

	if builderView, ok := view.(*tui.WorkflowBuilderView); ok {
		builderView.SetWorkflow(workflowName)
	}

	// Switch to builder view
	if err := app.GetViewManager().SwitchTo("builder"); err != nil {
		return fmt.Errorf("failed to switch to builder view: %w", err)
	}

	// Run the TUI
	if err := app.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}
