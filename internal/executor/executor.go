package executor

import (
	"context"
	"time"
)

// Result holds the outcome of a Claude Code execution.
type Result struct {
	SessionID string
	Output    string
	CostUSD   float64
	Turns     int
	Duration  time.Duration
	ExitCode  int
}

// Request holds parameters for a Claude Code execution.
type Request struct {
	TaskID         string
	Prompt         string
	ProjectPath    string
	SessionID      string
	Model          string
	AllowedTools   []string
	TimeoutMinutes int
	DryRun         bool
	Env            map[string]string
}

// ProgressFunc is called during execution to report progress.
type ProgressFunc func(eventType string, message string)

// Executor runs Claude Code tasks.
type Executor interface {
	Execute(ctx context.Context, req Request, onProgress ProgressFunc) (*Result, error)
}
