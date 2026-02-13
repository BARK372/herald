package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/btouchard/herald/internal/config"
)

func TestNewManager_SetsDefaults(t *testing.T) {
	t.Parallel()

	pm := NewManager(map[string]config.Project{
		"test": {
			Path: "/tmp",
		},
	})

	p, err := pm.Get("test")
	require.NoError(t, err)
	assert.Equal(t, 1, p.MaxConcurrentTasks)
	assert.Equal(t, "herald/", p.Git.BranchPrefix)
}

func TestManager_Validate_RejectsNonexistentPath(t *testing.T) {
	t.Parallel()

	pm := NewManager(map[string]config.Project{
		"missing": {
			Path: "/nonexistent/path/herald-test",
		},
	})

	err := pm.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestManager_Validate_RejectsFilePath(t *testing.T) {
	t.Parallel()

	tmpFile := filepath.Join(t.TempDir(), "notadir")
	require.NoError(t, os.WriteFile(tmpFile, []byte("x"), 0600))

	pm := NewManager(map[string]config.Project{
		"bad": {
			Path: tmpFile,
		},
	})

	err := pm.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a directory")
}

func TestManager_Validate_AcceptsValidPath(t *testing.T) {
	t.Parallel()

	pm := NewManager(map[string]config.Project{
		"valid": {
			Path: t.TempDir(),
		},
	})

	require.NoError(t, pm.Validate())
}

func TestManager_Get_ReturnsErrorForUnknown(t *testing.T) {
	t.Parallel()

	pm := NewManager(map[string]config.Project{})

	_, err := pm.Get("nope")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestManager_Default_ReturnsMarkedDefault(t *testing.T) {
	t.Parallel()

	pm := NewManager(map[string]config.Project{
		"a": {Path: "/tmp", Default: false},
		"b": {Path: "/tmp", Default: true},
	})

	p, err := pm.Default()
	require.NoError(t, err)
	assert.Equal(t, "b", p.Name)
}

func TestManager_Default_ReturnsSingleProject(t *testing.T) {
	t.Parallel()

	pm := NewManager(map[string]config.Project{
		"only": {Path: "/tmp"},
	})

	p, err := pm.Default()
	require.NoError(t, err)
	assert.Equal(t, "only", p.Name)
}

func TestManager_Default_ErrorsWhenAmbiguous(t *testing.T) {
	t.Parallel()

	pm := NewManager(map[string]config.Project{
		"a": {Path: "/tmp"},
		"b": {Path: "/tmp"},
	})

	_, err := pm.Default()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no default")
}

func TestManager_Resolve_UsesDefaultWhenEmpty(t *testing.T) {
	t.Parallel()

	pm := NewManager(map[string]config.Project{
		"myproj": {Path: "/tmp", Default: true},
	})

	p, err := pm.Resolve("")
	require.NoError(t, err)
	assert.Equal(t, "myproj", p.Name)
}

func TestManager_Resolve_UsesNameWhenProvided(t *testing.T) {
	t.Parallel()

	pm := NewManager(map[string]config.Project{
		"a": {Path: "/tmp", Default: true},
		"b": {Path: "/tmp"},
	})

	p, err := pm.Resolve("b")
	require.NoError(t, err)
	assert.Equal(t, "b", p.Name)
}

func TestManager_All_ReturnsAllProjects(t *testing.T) {
	t.Parallel()

	pm := NewManager(map[string]config.Project{
		"x": {Path: "/tmp"},
		"y": {Path: "/tmp"},
		"z": {Path: "/tmp"},
	})

	all := pm.All()
	assert.Len(t, all, 3)
}
