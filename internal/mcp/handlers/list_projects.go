package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/btouchard/herald/internal/project"
)

// ListProjects returns a handler that lists all configured projects.
func ListProjects(pm *project.Manager) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		projects := pm.All()

		if len(projects) == 0 {
			return mcp.NewToolResultText("No projects configured. Add projects to your herald.yaml configuration."), nil
		}

		var b strings.Builder
		fmt.Fprintf(&b, "**%d project(s) configured**\n\n", len(projects))

		for _, p := range projects {
			defaultMark := ""
			if p.Default {
				defaultMark = " (default)"
			}

			branch := pm.GitBranch(p)
			clean := pm.GitClean(p)

			gitStatus := ""
			if branch != "" {
				statusIcon := "clean"
				if !clean {
					statusIcon = "dirty"
				}
				gitStatus = fmt.Sprintf(" | git: %s (%s)", branch, statusIcon)
			}

			fmt.Fprintf(&b, "**%s**%s\n", p.Name, defaultMark)
			if p.Description != "" {
				fmt.Fprintf(&b, "  %s\n", p.Description)
			}
			fmt.Fprintf(&b, "  Path: %s%s\n", p.Path, gitStatus)
			fmt.Fprintf(&b, "  Concurrency: %d task(s)\n", p.MaxConcurrentTasks)
			if len(p.AllowedTools) > 0 {
				fmt.Fprintf(&b, "  Tools: %s\n", strings.Join(p.AllowedTools, ", "))
			}
			b.WriteString("\n")
		}

		return mcp.NewToolResultText(b.String()), nil
	}
}
