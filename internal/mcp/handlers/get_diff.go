package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/kolapsis/herald/internal/git"
	"github.com/kolapsis/herald/internal/project"
	"github.com/kolapsis/herald/internal/task"
)

// GetDiff returns a handler that shows git diff.
// Accepts either task_id (to resolve project from task) or project directly.
func GetDiff(tm *task.Manager, pm *project.Manager) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		taskID, _ := args["task_id"].(string)
		projectName, _ := args["project"].(string)

		var proj *project.Project
		var taskBranch string
		var label string

		switch {
		case taskID != "":
			t, err := tm.Get(taskID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Task not found: %s", err)), nil
			}
			snap := t.Snapshot()
			taskBranch = snap.GitBranch
			proj, err = pm.Get(snap.Project)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Project not found: %s", err)), nil
			}
			label = fmt.Sprintf("task %s", taskID)

		case projectName != "":
			var err error
			proj, err = pm.Resolve(projectName)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Project error: %s", err)), nil
			}
			label = fmt.Sprintf("project %s", proj.Name)

		default:
			return mcp.NewToolResultError("task_id or project is required"), nil
		}

		ops := git.NewOps(proj.Path)
		if !ops.IsGitRepo(ctx) {
			return mcp.NewToolResultError(fmt.Sprintf("Project %q is not a git repository", proj.Name)), nil
		}

		if !ops.HasCommits(ctx) {
			return mcp.NewToolResultText(fmt.Sprintf("No changes detected for %s (repository has no commits yet).", label)), nil
		}

		var diff string
		var err error
		if taskBranch != "" {
			branch, brErr := ops.CurrentBranch(ctx)
			if brErr != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to get current branch: %s", brErr)), nil
			}
			diff, err = ops.Diff(ctx, branch, taskBranch)
		} else {
			diff, err = ops.Diff(ctx, "HEAD", "")
		}
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get diff: %s", err)), nil
		}

		if strings.TrimSpace(diff) == "" {
			return mcp.NewToolResultText(fmt.Sprintf("No changes detected for %s.", label)), nil
		}

		var sb strings.Builder
		fmt.Fprintf(&sb, "Diff for %s\n\n", label)
		sb.WriteString("```diff\n")
		sb.WriteString(diff)
		sb.WriteString("\n```\n")

		return mcp.NewToolResultText(sb.String()), nil
	}
}
