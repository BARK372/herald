package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaults_SetsExpectedValues(t *testing.T) {
	t.Parallel()

	cfg := Defaults()

	assert.Equal(t, "127.0.0.1", cfg.Server.Host)
	assert.Equal(t, 8420, cfg.Server.Port)
	assert.Equal(t, "info", cfg.Server.LogLevel)
	assert.Equal(t, "claude", cfg.Execution.ClaudePath)
	assert.Equal(t, 30*time.Minute, cfg.Execution.DefaultTimeout)
	assert.Equal(t, 3, cfg.Execution.MaxConcurrent)
	assert.Equal(t, 60, cfg.RateLimit.RequestsPerMinute)
	assert.True(t, cfg.Dashboard.Enabled)
	assert.Equal(t, time.Hour, cfg.Auth.AccessTokenTTL)
	assert.Equal(t, 30*24*time.Hour, cfg.Auth.RefreshTokenTTL)
}

func TestLoadFromFile_ParsesYAML(t *testing.T) {
	t.Parallel()

	content := `
server:
  host: "127.0.0.1"
  port: 9000
  public_url: "https://herald.test.com"
  log_level: "debug"

execution:
  claude_path: "/usr/local/bin/claude"
  default_timeout: 15m
  max_concurrent: 2

projects:
  test-project:
    path: "/tmp/test-project"
    description: "A test project"
    default: true
    allowed_tools:
      - "Read"
      - "Write"
    max_concurrent_tasks: 1
    git:
      auto_branch: true
      branch_prefix: "herald/"
`

	tmpFile := filepath.Join(t.TempDir(), "herald.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0644))

	cfg, err := LoadFromFile(tmpFile)
	require.NoError(t, err)

	assert.Equal(t, 9000, cfg.Server.Port)
	assert.Equal(t, "https://herald.test.com", cfg.Server.PublicURL)
	assert.Equal(t, "debug", cfg.Server.LogLevel)
	assert.Equal(t, "/usr/local/bin/claude", cfg.Execution.ClaudePath)
	assert.Equal(t, 15*time.Minute, cfg.Execution.DefaultTimeout)
	assert.Equal(t, 2, cfg.Execution.MaxConcurrent)

	require.Contains(t, cfg.Projects, "test-project")
	proj := cfg.Projects["test-project"]
	assert.Equal(t, "/tmp/test-project", proj.Path)
	assert.True(t, proj.Default)
	assert.Equal(t, []string{"Read", "Write"}, proj.AllowedTools)
	assert.True(t, proj.Git.AutoBranch)
	assert.Equal(t, "herald/", proj.Git.BranchPrefix)
}

func TestLoadFromFile_ExpandsEnvVars(t *testing.T) {
	t.Setenv("HERALD_TEST_SECRET", "super-secret-value")

	content := `
auth:
  client_secret: "${HERALD_TEST_SECRET}"
`
	tmpFile := filepath.Join(t.TempDir(), "herald.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0644))

	cfg, err := LoadFromFile(tmpFile)
	require.NoError(t, err)

	assert.Equal(t, "super-secret-value", cfg.Auth.ClientSecret)
}

func TestLoadFromFile_RejectsBindAllInterfaces(t *testing.T) {
	t.Parallel()

	content := `
server:
  host: "0.0.0.0"
  port: 8420
`
	tmpFile := filepath.Join(t.TempDir(), "herald.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0644))

	_, err := LoadFromFile(tmpFile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "0.0.0.0")
}

func TestLoadFromFile_RejectsInvalidPort(t *testing.T) {
	t.Parallel()

	content := `
server:
  port: 99999
`
	tmpFile := filepath.Join(t.TempDir(), "herald.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0644))

	_, err := LoadFromFile(tmpFile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "port")
}

func TestLoadFromFile_NonexistentFileReturnsDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := LoadFromFile("/tmp/herald-nonexistent-config-file.yaml")
	require.NoError(t, err)

	assert.Equal(t, 8420, cfg.Server.Port)
	assert.Equal(t, "127.0.0.1", cfg.Server.Host)
}

func TestExpandHome_ReplacesLeadingTilde(t *testing.T) {
	t.Parallel()

	home, err := os.UserHomeDir()
	require.NoError(t, err)

	result := ExpandHome("~/some/path")
	assert.Equal(t, filepath.Join(home, "some/path"), result)
}

func TestExpandHome_LeavesAbsolutePathsUnchanged(t *testing.T) {
	t.Parallel()

	result := ExpandHome("/absolute/path")
	assert.Equal(t, "/absolute/path", result)
}
