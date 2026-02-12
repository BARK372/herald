package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/kolapsis/herald/internal/task"
)

// ListTasks returns a handler that lists tasks with optional filters.
func ListTasks(tm *task.Manager) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		filter := task.Filter{
			Limit: 20,
		}

		if status, ok := args["status"].(string); ok {
			filter.Status = status
		}
		if project, ok := args["project"].(string); ok {
			filter.Project = project
		}
		if limit, ok := args["limit"].(float64); ok && limit > 0 {
			filter.Limit = int(limit)
		}

		tasks := tm.List(filter)

		if len(tasks) == 0 {
			return mcp.NewToolResultText("No tasks found matching the given filters."), nil
		}

		var sb strings.Builder
		fmt.Fprintf(&sb, "📋 Tasks (%d found)\n\n", len(tasks))

		for _, t := range tasks {
			icon := statusIcon(t.Status)
			sb.WriteString(fmt.Sprintf("%s **%s** — %s\n", icon, t.ID, t.Status))
			sb.WriteString(fmt.Sprintf("  Project: %s | Priority: %s\n", t.Project, t.Priority))

			if t.Status == task.StatusRunning {
				sb.WriteString(fmt.Sprintf("  Duration: %s", t.FormatDuration()))
				if t.Progress != "" {
					sb.WriteString(fmt.Sprintf(" | Progress: %s", t.Progress))
				}
				sb.WriteString("\n")
			}

			if t.Status == task.StatusCompleted || t.Status == task.StatusFailed {
				sb.WriteString(fmt.Sprintf("  Duration: %s | Cost: $%.2f\n", t.FormatDuration(), t.CostUSD))
			}

			if t.Error != "" {
				sb.WriteString(fmt.Sprintf("  Error: %s\n", t.Error))
			}

			sb.WriteString("\n")
		}

		return mcp.NewToolResultText(sb.String()), nil
	}
}

func statusIcon(s task.Status) string {
	switch s {
	case task.StatusPending:
		return "⏳"
	case task.StatusQueued:
		return "📥"
	case task.StatusRunning:
		return "🔄"
	case task.StatusCompleted:
		return "✅"
	case task.StatusFailed:
		return "❌"
	case task.StatusCancelled:
		return "🚫"
	default:
		return "❓"
	}
}
