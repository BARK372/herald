package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/btouchard/herald/internal/task"
)

// GetLogs returns a handler that shows task activity and events.
func GetLogs(tm *task.Manager) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		taskID, _ := args["task_id"].(string)
		limit := 20
		if l, ok := args["limit"].(float64); ok && l > 0 {
			limit = int(l)
		}

		if taskID != "" {
			return getTaskLogs(tm, taskID)
		}

		return getRecentActivity(tm, limit)
	}
}

func getTaskLogs(tm *task.Manager, taskID string) (*mcp.CallToolResult, error) {
	t, err := tm.Get(taskID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Task not found: %s", err)), nil
	}

	snap := t.Snapshot()

	var sb strings.Builder
	fmt.Fprintf(&sb, "📋 Logs for task %s\n\n", taskID)
	fmt.Fprintf(&sb, "Status: %s %s\n", statusIcon(snap.Status), snap.Status)
	fmt.Fprintf(&sb, "Project: %s\n", snap.Project)
	fmt.Fprintf(&sb, "Created: %s\n", snap.CreatedAt.Format("2006-01-02 15:04:05"))

	if !snap.StartedAt.IsZero() {
		fmt.Fprintf(&sb, "Started: %s\n", snap.StartedAt.Format("2006-01-02 15:04:05"))
	}
	if !snap.CompletedAt.IsZero() {
		fmt.Fprintf(&sb, "Completed: %s\n", snap.CompletedAt.Format("2006-01-02 15:04:05"))
	}

	fmt.Fprintf(&sb, "Duration: %s\n", snap.FormatDuration())

	if snap.SessionID != "" {
		fmt.Fprintf(&sb, "Session: %s\n", snap.SessionID)
	}
	if snap.CostUSD > 0 {
		fmt.Fprintf(&sb, "Cost: $%.4f\n", snap.CostUSD)
	}
	if snap.Turns > 0 {
		fmt.Fprintf(&sb, "Turns: %d\n", snap.Turns)
	}
	if snap.Error != "" {
		fmt.Fprintf(&sb, "\nError: %s\n", snap.Error)
	}
	if snap.Progress != "" {
		fmt.Fprintf(&sb, "\nLast progress: %s\n", snap.Progress)
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func getRecentActivity(tm *task.Manager, limit int) (*mcp.CallToolResult, error) {
	tasks := tm.List(task.Filter{Status: "all", Limit: limit})

	if len(tasks) == 0 {
		return mcp.NewToolResultText("No activity recorded yet."), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "📋 Recent activity (%d tasks)\n\n", len(tasks))

	for _, t := range tasks {
		icon := statusIcon(t.Status)
		fmt.Fprintf(&sb, "%s %s — %s (%s) — %s\n",
			icon, t.ID, t.Status, t.Project, t.CreatedAt.Format("15:04:05"))
	}

	return mcp.NewToolResultText(sb.String()), nil
}
