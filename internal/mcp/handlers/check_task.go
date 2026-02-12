package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/kolapsis/herald/internal/task"
)

// CheckTask returns a handler that reports a task's current status.
func CheckTask(tm *task.Manager) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		taskID, _ := args["task_id"].(string)
		if taskID == "" {
			return mcp.NewToolResultError("task_id is required"), nil
		}

		t, err := tm.Get(taskID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Task not found: %s", err)), nil
		}

		snap := t.Snapshot()
		includeOutput, _ := args["include_output"].(bool)
		outputLines := 20
		if n, ok := args["output_lines"].(float64); ok && n > 0 {
			outputLines = int(n)
		}

		var b strings.Builder

		switch snap.Status {
		case task.StatusPending, task.StatusQueued:
			fmt.Fprintf(&b, "Status: %s\n", snap.Status)

		case task.StatusRunning:
			fmt.Fprintf(&b, "Status: running\n")
			fmt.Fprintf(&b, "Duration: %s\n", snap.FormatDuration())
			if snap.Progress != "" {
				fmt.Fprintf(&b, "Progress: %s\n", snap.Progress)
			}
			if snap.CostUSD > 0 {
				fmt.Fprintf(&b, "Cost so far: ~$%.2f\n", snap.CostUSD)
			}

		case task.StatusCompleted:
			fmt.Fprintf(&b, "Status: completed\n")
			fmt.Fprintf(&b, "Duration: %s\n", snap.FormatDuration())
			if snap.CostUSD > 0 {
				fmt.Fprintf(&b, "Cost: $%.2f\n", snap.CostUSD)
			}
			if snap.Turns > 0 {
				fmt.Fprintf(&b, "Turns: %d\n", snap.Turns)
			}
			if snap.SessionID != "" {
				fmt.Fprintf(&b, "Session ID: %s (use to continue this conversation)\n", snap.SessionID)
			}
			b.WriteString("\nUse get_result for full output, get_diff for changes.")

		case task.StatusFailed:
			fmt.Fprintf(&b, "Status: failed\n")
			fmt.Fprintf(&b, "Duration: %s\n", snap.FormatDuration())
			if snap.Error != "" {
				fmt.Fprintf(&b, "Error: %s\n", snap.Error)
			}

		case task.StatusCancelled:
			fmt.Fprintf(&b, "Status: cancelled\n")
			fmt.Fprintf(&b, "Duration: %s\n", snap.FormatDuration())
		}

		if includeOutput && snap.Output != "" {
			lines := lastNLines(snap.Output, outputLines)
			fmt.Fprintf(&b, "\n--- Last output ---\n%s", lines)
		}

		return mcp.NewToolResultText(b.String()), nil
	}
}

func lastNLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}
