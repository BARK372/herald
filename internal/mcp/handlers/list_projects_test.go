package handlers

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kolapsis/herald/internal/config"
	"github.com/kolapsis/herald/internal/project"
)

func TestListProjects_WhenNoProjects_ReturnsMessage(t *testing.T) {
	t.Parallel()

	pm := project.NewManager(map[string]config.Project{})
	handler := ListProjects(pm)

	result, err := handler(context.Background(), mcp.CallToolRequest{})
	require.NoError(t, err)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(mcp.TextContent).Text
	assert.Contains(t, text, "No projects configured")
}

func TestListProjects_WhenProjectsExist_ReturnsFormatted(t *testing.T) {
	t.Parallel()

	pm := project.NewManager(map[string]config.Project{
		"my-api": {
			Path:               "/tmp",
			Description:        "Backend API",
			Default:            true,
			AllowedTools:       []string{"Read", "Write"},
			MaxConcurrentTasks: 2,
		},
	})

	handler := ListProjects(pm)

	result, err := handler(context.Background(), mcp.CallToolRequest{})
	require.NoError(t, err)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(mcp.TextContent).Text
	assert.Contains(t, text, "1 project(s) configured")
	assert.Contains(t, text, "my-api")
	assert.Contains(t, text, "(default)")
	assert.Contains(t, text, "Backend API")
	assert.Contains(t, text, "Read, Write")
	assert.Contains(t, text, "2 task(s)")
}
