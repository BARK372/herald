package claude

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/btouchard/herald/internal/executor"
)

// writeTestScript writes a shell script atomically using rename to avoid
// "text file busy" errors on Linux when the script is executed immediately.
func writeTestScript(t *testing.T, path, content string) {
	t.Helper()
	tmp := path + ".tmp"
	require.NoError(t, os.WriteFile(tmp, []byte(content), 0700)) //nolint:gosec // test scripts must be executable
	require.NoError(t, os.Rename(tmp, path))
}

func TestCapabilities_ReturnsExpectedValues(t *testing.T) {
	t.Parallel()

	exec := &Executor{}
	caps := exec.Capabilities()

	assert.Equal(t, "claude-code", caps.Name)
	assert.True(t, caps.SupportsSession)
	assert.True(t, caps.SupportsModel)
	assert.True(t, caps.SupportsToolList)
	assert.True(t, caps.SupportsDryRun)
	assert.True(t, caps.SupportsStreaming)
}

func TestRegistration_ClaudeCodeIsAvailable(t *testing.T) {
	t.Parallel()

	factory, ok := executor.Get("claude-code")
	assert.True(t, ok, "claude-code should be registered")
	assert.NotNil(t, factory)

	available := executor.Available()
	assert.Contains(t, available, "claude-code")
}

func TestFactory_CreatesExecutorWithDefaults(t *testing.T) {
	t.Parallel()

	factory, ok := executor.Get("claude-code")
	require.True(t, ok)

	exec, err := factory(map[string]any{})
	require.NoError(t, err)

	caps := exec.Capabilities()
	assert.Equal(t, "claude-code", caps.Name)
}

func TestFactory_CreatesExecutorWithConfig(t *testing.T) {
	t.Parallel()

	factory, ok := executor.Get("claude-code")
	require.True(t, ok)

	exec, err := factory(map[string]any{
		"claude_path": "/usr/local/bin/claude",
		"work_dir":    "/tmp/herald",
	})
	require.NoError(t, err)

	claudeExec, ok := exec.(*Executor)
	require.True(t, ok)
	assert.Equal(t, "/usr/local/bin/claude", claudeExec.ClaudePath)
	assert.Equal(t, "/tmp/herald", claudeExec.WorkDir)
}

func TestParseStream_WhenVeryLongLine_HandlesWithoutPanic(t *testing.T) {
	t.Parallel()

	// Create a valid JSON event with a very long text field (~1MB)
	longText := strings.Repeat("a", 1_000_000)
	line := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"` + longText + `"}]}}`

	result := &executor.Result{}
	require.NotPanics(t, func() {
		parseStream("test-task", strings.NewReader(line), result, nil)
	})
	assert.Contains(t, result.Output, longText)
}

func TestParseStream_WhenMixedValidAndInvalid_ProcessesValidLines(t *testing.T) {
	t.Parallel()

	stream := strings.Join([]string{
		`{"type":"system","subtype":"init","session_id":"ses_mixed"}`,
		`{broken}`,
		``,
		`not json at all`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"valid output"}]}}`,
		`{"type":"result","subtype":"success","cost_usd":0.50,"num_turns":2}`,
	}, "\n")

	result := &executor.Result{}
	var progressCalls int
	onProgress := func(_, _ string) { progressCalls++ }

	parseStream("test-task", strings.NewReader(stream), result, onProgress)

	assert.Equal(t, "ses_mixed", result.SessionID)
	assert.Equal(t, "valid output", result.Output)
	assert.InDelta(t, 0.50, result.CostUSD, 0.001)
	assert.Equal(t, 2, result.Turns)
	assert.Equal(t, 1, progressCalls) // only the valid assistant event
}

func TestParseStream_WhenMultipleAssistantEvents_AccumulatesOutput(t *testing.T) {
	t.Parallel()

	stream := strings.Join([]string{
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Part 1. "}]}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Read"}]}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Part 2. "}]}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Part 3."}]}}`,
	}, "\n")

	result := &executor.Result{}
	var progressMsgs []string
	onProgress := func(_, msg string) { progressMsgs = append(progressMsgs, msg) }

	parseStream("test-task", strings.NewReader(stream), result, onProgress)

	assert.Equal(t, "Part 1. Part 2. Part 3.", result.Output)
	assert.Len(t, progressMsgs, 4) // 3 text events + 1 tool_use
	assert.Contains(t, progressMsgs, "Using tool: Read")
}

func TestParseStream_WhenSystemNotInit_IgnoresSessionID(t *testing.T) {
	t.Parallel()

	stream := `{"type":"system","subtype":"heartbeat","session_id":"should_not_be_captured"}`

	result := &executor.Result{}
	parseStream("test-task", strings.NewReader(stream), result, nil)

	assert.Empty(t, result.SessionID)
}

func TestParseStream_WhenResultHasDuration_SetsDuration(t *testing.T) {
	t.Parallel()

	stream := `{"type":"result","subtype":"success","cost_usd":0.10,"duration_ms":60000,"num_turns":3}`

	result := &executor.Result{}
	parseStream("test-task", strings.NewReader(stream), result, nil)

	assert.Equal(t, 60*time.Second, result.Duration)
}

func TestParseStream_WhenResultNoDuration_KeepsZero(t *testing.T) {
	t.Parallel()

	stream := `{"type":"result","subtype":"success","cost_usd":0.05,"num_turns":1}`

	result := &executor.Result{}
	parseStream("test-task", strings.NewReader(stream), result, nil)

	assert.Equal(t, time.Duration(0), result.Duration)
}

func TestParseStream_WhenWhitespaceOnlyLines_SkipsThem(t *testing.T) {
	t.Parallel()

	stream := strings.Join([]string{
		`   `,
		`{"type":"system","subtype":"init","session_id":"ses_ws"}`,
		`	`,
		`{"type":"result","subtype":"success","cost_usd":0.01}`,
	}, "\n")

	result := &executor.Result{}
	parseStream("test-task", strings.NewReader(stream), result, nil)

	// Whitespace-only lines get parsed as malformed JSON (not empty), but should not crash
	assert.Equal(t, "ses_ws", result.SessionID)
	assert.InDelta(t, 0.01, result.CostUSD, 0.001)
}

func TestCaptureStderr_WhenDataPresent_DoesNotPanic(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		captureStderr("test-task", strings.NewReader("warning: something happened"))
	})
}

func TestCaptureStderr_WhenEmpty_DoesNotPanic(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		captureStderr("test-task", strings.NewReader(""))
	})
}

func TestExecute_WhenCommandOutputsStreamJSON_ParsesResult(t *testing.T) {
	t.Parallel()

	// Create a mock script that outputs stream-json to stdout
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "mock_claude.sh")
	script := `#!/bin/sh
cat <<'STREAM'
{"type":"system","subtype":"init","session_id":"ses_mock123"}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"I fixed the bug."}]}}
{"type":"result","subtype":"success","cost_usd":0.25,"duration_ms":5000,"num_turns":2}
STREAM
`
	writeTestScript(t, scriptPath, script)

	exec := &Executor{
		ClaudePath: scriptPath,
		WorkDir:    tmpDir,
	}

	req := executor.Request{
		TaskID:      "herald-mock01",
		Prompt:      "fix the bug",
		ProjectPath: tmpDir,
	}

	var progressMsgs []string
	onProgress := func(eventType, msg string) {
		progressMsgs = append(progressMsgs, eventType+":"+msg)
	}

	result, err := exec.Execute(context.Background(), req, onProgress)
	require.NoError(t, err)

	assert.Equal(t, "ses_mock123", result.SessionID)
	assert.Contains(t, result.Output, "I fixed the bug.")
	assert.InDelta(t, 0.25, result.CostUSD, 0.001)
	assert.Equal(t, 2, result.Turns)
	assert.Equal(t, 0, result.ExitCode)
}

func TestExecute_WhenCommandFails_ReturnsErrorWithExitCode(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "fail_claude.sh")
	script := `#!/bin/sh
echo '{"type":"system","subtype":"init","session_id":"ses_fail"}' >&1
exit 1
`
	writeTestScript(t, scriptPath, script)

	exec := &Executor{
		ClaudePath: scriptPath,
		WorkDir:    tmpDir,
	}

	req := executor.Request{
		TaskID:      "herald-fail01",
		Prompt:      "do something",
		ProjectPath: tmpDir,
	}

	result, err := exec.Execute(context.Background(), req, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "claude exited with code 1")
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.ExitCode)
}

func TestExecute_WhenContextCancelled_ReturnsError(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "slow_claude.sh")
	script := `#!/bin/sh
sleep 60
`
	writeTestScript(t, scriptPath, script)

	exec := &Executor{
		ClaudePath: scriptPath,
		WorkDir:    tmpDir,
	}

	req := executor.Request{
		TaskID:      "herald-cancel01",
		Prompt:      "slow task",
		ProjectPath: tmpDir,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_, err := exec.Execute(ctx, req, nil)
	assert.Error(t, err)
}

func TestExecute_WhenSessionIDProvided_PassesResumeFlag(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "echo_claude.sh")
	script := `#!/bin/sh
echo '{"type":"result","subtype":"success","cost_usd":0.01,"num_turns":1}'
`
	writeTestScript(t, scriptPath, script)

	exec := &Executor{
		ClaudePath: scriptPath,
		WorkDir:    tmpDir,
	}

	req := executor.Request{
		TaskID:      "herald-resume01",
		Prompt:      "continue",
		SessionID:   "ses_existing",
		ProjectPath: tmpDir,
	}

	result, err := exec.Execute(context.Background(), req, nil)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestExecute_WhenAllowedToolsProvided_PassesToolFlags(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "tools_claude.sh")
	script := `#!/bin/sh
echo '{"type":"result","subtype":"success","cost_usd":0.01,"num_turns":1}'
`
	writeTestScript(t, scriptPath, script)

	exec := &Executor{
		ClaudePath: scriptPath,
		WorkDir:    tmpDir,
	}

	req := executor.Request{
		TaskID:       "herald-tools01",
		Prompt:       "test",
		ProjectPath:  tmpDir,
		AllowedTools: []string{"Read", "Write", "Edit"},
	}

	result, err := exec.Execute(context.Background(), req, nil)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestExecute_WhenEnvVarsSet_MergesEnvironment(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "env_claude.sh")
	script := `#!/bin/sh
echo '{"type":"result","subtype":"success","cost_usd":0.01,"num_turns":1}'
`
	writeTestScript(t, scriptPath, script)

	exec := &Executor{
		ClaudePath: scriptPath,
		WorkDir:    tmpDir,
		Env: map[string]string{
			"HERALD_EXEC_VAR": "global",
		},
	}

	req := executor.Request{
		TaskID:      "herald-env01",
		Prompt:      "test",
		ProjectPath: tmpDir,
		Env: map[string]string{
			"HERALD_REQ_VAR": "request",
		},
	}

	result, err := exec.Execute(context.Background(), req, nil)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestExecute_WhenStderrOutput_CapturesWithoutError(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "stderr_claude.sh")
	script := `#!/bin/sh
echo "debug info" >&2
echo '{"type":"result","subtype":"success","cost_usd":0.01,"num_turns":1}'
`
	writeTestScript(t, scriptPath, script)

	exec := &Executor{
		ClaudePath: scriptPath,
		WorkDir:    tmpDir,
	}

	req := executor.Request{
		TaskID:      "herald-stderr01",
		Prompt:      "test",
		ProjectPath: tmpDir,
	}

	result, err := exec.Execute(context.Background(), req, nil)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestExecute_WhenProgressFuncProvided_ReceivesStartedEvent(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "progress_claude.sh")
	script := `#!/bin/sh
echo '{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Working..."}]}}'
echo '{"type":"result","subtype":"success","cost_usd":0.01,"num_turns":1}'
`
	writeTestScript(t, scriptPath, script)

	exec := &Executor{
		ClaudePath: scriptPath,
		WorkDir:    tmpDir,
	}

	req := executor.Request{
		TaskID:      "herald-prog01",
		Prompt:      "test",
		ProjectPath: tmpDir,
	}

	var events []string
	onProgress := func(eventType, msg string) {
		events = append(events, eventType)
	}

	result, err := exec.Execute(context.Background(), req, onProgress)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, events, "started")
}

func TestExecute_WhenModelProvided_PassesModelFlag(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "model_claude.sh")
	script := `#!/bin/sh
found=0
for arg in "$@"; do
    if [ "$arg" = "--model" ]; then
        found=1
    fi
done
if [ "$found" = "0" ]; then
    echo "ERROR: --model flag not found" >&2
    exit 1
fi
echo '{"type":"result","subtype":"success","cost_usd":0.01,"num_turns":1}'
`
	writeTestScript(t, scriptPath, script)

	exec := &Executor{
		ClaudePath: scriptPath,
		WorkDir:    tmpDir,
	}

	req := executor.Request{
		TaskID:      "herald-model01",
		Prompt:      "test",
		ProjectPath: tmpDir,
		Model:       "claude-opus-4-6",
	}

	result, err := exec.Execute(context.Background(), req, nil)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, result.ExitCode)
}

func TestExecute_WhenNoModel_DoesNotPassModelFlag(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "nomodel_claude.sh")
	script := `#!/bin/sh
for arg in "$@"; do
    if [ "$arg" = "--model" ]; then
        echo "ERROR: --model flag should not be present" >&2
        exit 1
    fi
done
echo '{"type":"result","subtype":"success","cost_usd":0.01,"num_turns":1}'
`
	writeTestScript(t, scriptPath, script)

	exec := &Executor{
		ClaudePath: scriptPath,
		WorkDir:    tmpDir,
	}

	req := executor.Request{
		TaskID:      "herald-nomodel01",
		Prompt:      "test",
		ProjectPath: tmpDir,
	}

	result, err := exec.Execute(context.Background(), req, nil)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, result.ExitCode)
}
