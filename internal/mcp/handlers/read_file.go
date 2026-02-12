package handlers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/kolapsis/herald/internal/project"
)

const maxFileSize = 1024 * 1024 // 1MB

// ReadFile returns a handler that reads a file from a project with path traversal prevention.
func ReadFile(pm *project.Manager) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		projectName, _ := args["project"].(string)
		filePath, ok := args["path"].(string)
		if !ok || filePath == "" {
			return mcp.NewToolResultError("path is required"), nil
		}

		proj, err := pm.Resolve(projectName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Project error: %s", err)), nil
		}

		safePath, err := SafePath(proj.Path, filePath)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Access denied: %s", err)), nil
		}

		info, err := os.Stat(safePath)
		if err != nil {
			if os.IsNotExist(err) {
				return mcp.NewToolResultError(fmt.Sprintf("File not found: %s", filePath)), nil
			}
			return mcp.NewToolResultError(fmt.Sprintf("Cannot access file: %s", err)), nil
		}

		if info.IsDir() {
			return mcp.NewToolResultError(fmt.Sprintf("%s is a directory, not a file", filePath)), nil
		}

		if info.Size() > maxFileSize {
			return mcp.NewToolResultError(fmt.Sprintf("File too large (%d bytes, max %d)", info.Size(), maxFileSize)), nil
		}

		content, err := os.ReadFile(safePath)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to read file: %s", err)), nil
		}

		var sb strings.Builder
		fmt.Fprintf(&sb, "📄 %s (%d bytes)\n\n", filePath, len(content))
		sb.WriteString("```\n")
		sb.Write(content)
		sb.WriteString("\n```\n")

		return mcp.NewToolResultText(sb.String()), nil
	}
}

// SafePath validates that the requested path stays within the project root.
// This prevents path traversal attacks (e.g., ../../etc/passwd).
func SafePath(projectRoot, requestedPath string) (string, error) {
	// Reject absolute paths immediately
	if filepath.IsAbs(requestedPath) {
		return "", fmt.Errorf("absolute paths are not allowed: %s", requestedPath)
	}

	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return "", fmt.Errorf("resolving project root: %w", err)
	}

	// Join and resolve to absolute path (resolves .., etc.)
	absPath, err := filepath.Abs(filepath.Join(projectRoot, requestedPath))
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}

	// Ensure the resolved path is within the project root
	if !strings.HasPrefix(absPath, absRoot+string(filepath.Separator)) && absPath != absRoot {
		return "", fmt.Errorf("path traversal detected: %s resolves outside project root", requestedPath)
	}

	return absPath, nil
}
