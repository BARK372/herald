package executor

import (
	"context"
	"time"
)

// Result holds the outcome of a task execution.
type Result struct {
	SessionID string
	Output    string
	CostUSD   float64
	Turns     int
	Duration  time.Duration
	ExitCode  int
}

// Request holds parameters for a task execution.
// Fields that require specific capabilities (see Capabilities) are silently
// ignored by executors that do not support them.
type Request struct {
	TaskID      string
	Prompt      string
	ProjectPath string

	// SessionID resumes a previous conversation. Requires Capabilities.SupportsSession.
	SessionID string

	// Model overrides the default model. Requires Capabilities.SupportsModel.
	Model string

	// AllowedTools restricts which tools the executor can use. Requires Capabilities.SupportsToolList.
	AllowedTools []string

	TimeoutMinutes int

	// DryRun requests planning without execution. Requires Capabilities.SupportsDryRun.
	DryRun bool

	Env map[string]string
}

// ProgressFunc is called during execution to report progress.
type ProgressFunc func(eventType string, message string)

// Capabilities describes what features an executor implementation supports.
// Handlers use this to warn users when a requested feature is unavailable.
type Capabilities struct {
	SupportsSession  bool
	SupportsModel    bool
	SupportsToolList bool
	SupportsDryRun   bool
	SupportsStreaming bool
	Name             string
	Version          string
}

// Executor runs tasks against a CLI backend.
type Executor interface {
	Execute(ctx context.Context, req Request, onProgress ProgressFunc) (*Result, error)
	Capabilities() Capabilities
}
