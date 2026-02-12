package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/kolapsis/herald/internal/task"
)

// HeraldPush returns a handler that registers a Claude Code session as a linked task.
// If a linked task with the same session_id already exists, it is updated instead.
func HeraldPush(tm *task.Manager) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		sessionID, _ := args["session_id"].(string)
		if sessionID == "" {
			return mcp.NewToolResultError("session_id is required"), nil
		}

		summary, _ := args["summary"].(string)
		if summary == "" {
			return mcp.NewToolResultError("summary is required"), nil
		}

		projectName, _ := args["project"].(string)
		currentTask, _ := args["current_task"].(string)
		gitBranch, _ := args["git_branch"].(string)

		var turns int
		if t, ok := args["turns"].(float64); ok && t > 0 {
			turns = int(t)
		}

		var filesModified []string
		if files, ok := args["files_modified"].([]interface{}); ok {
			for _, f := range files {
				if s, ok := f.(string); ok {
					filesModified = append(filesModified, s)
				}
			}
		}

		// Check for existing linked task with same session_id — update instead of duplicate
		if existing := tm.GetBySessionID(sessionID, task.StatusLinked); existing != nil {
			existing.SetOutput(summary)
			existing.SetLinkedFields(projectName, gitBranch, currentTask, turns, filesModified)

			return mcp.NewToolResultText(formatPushResponse(existing.ID, sessionID, projectName, true)), nil
		}

		// Create new linked task
		t := task.NewLinked(sessionID, projectName, summary, currentTask, gitBranch, turns, filesModified)
		tm.Register(t)

		return mcp.NewToolResultText(formatPushResponse(t.ID, sessionID, projectName, false)), nil
	}
}

func formatPushResponse(taskID, sessionID, project string, updated bool) string {
	var b strings.Builder

	if updated {
		fmt.Fprintf(&b, "Session updated in Herald\n\n")
	} else {
		fmt.Fprintf(&b, "Session pushed to Herald\n\n")
	}

	fmt.Fprintf(&b, "- Task ID: %s\n", taskID)
	fmt.Fprintf(&b, "- Session: %s\n", sessionID)
	if project != "" {
		fmt.Fprintf(&b, "- Project: %s\n", project)
	}
	fmt.Fprintf(&b, "- Status: linked\n\n")
	fmt.Fprintf(&b, "You can now continue this session from Claude Chat:\n")
	fmt.Fprintf(&b, "  list_tasks to find it\n")
	fmt.Fprintf(&b, "  check_task for the full summary\n")
	fmt.Fprintf(&b, "  start_task with session_id %q to resume", sessionID)

	return b.String()
}
