package handlers

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/btouchard/herald/internal/task"
)

// CancelTask returns a handler that cancels a running task.
func CancelTask(tm *task.Manager) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		taskID, ok := args["task_id"].(string)
		if !ok || taskID == "" {
			return mcp.NewToolResultError("task_id is required"), nil
		}

		if err := tm.Cancel(taskID); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to cancel task: %s", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("🚫 Task %s has been cancelled.", taskID)), nil
	}
}
