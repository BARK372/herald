package executor

import (
	"fmt"
	"os"
	"path/filepath"
)

// WritePromptFile writes the prompt to a temp file and returns its path.
// Claude Code prompts are piped via stdin from this file to avoid
// CLI argument length limits (~7000 chars).
func WritePromptFile(workDir, taskID, prompt string) (string, error) {
	dir := filepath.Join(workDir, "tasks", taskID)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return "", fmt.Errorf("creating prompt directory: %w", err)
	}

	path := filepath.Join(dir, "prompt.md")
	if err := os.WriteFile(path, []byte(prompt), 0640); err != nil {
		return "", fmt.Errorf("writing prompt file: %w", err)
	}

	return path, nil
}

// CleanupPromptFile removes the prompt temp directory for a task.
func CleanupPromptFile(workDir, taskID string) {
	dir := filepath.Join(workDir, "tasks", taskID)
	_ = os.RemoveAll(dir)
}
