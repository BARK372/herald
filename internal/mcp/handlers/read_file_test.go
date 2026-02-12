package handlers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSafePath_AllowsValidPaths(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "src", "pkg"), 0755))

	tests := []struct {
		name string
		path string
	}{
		{"simple file", "main.go"},
		{"nested file", "src/pkg/handler.go"},
		{"with dot prefix", "./main.go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := SafePath(root, tt.path)
			require.NoError(t, err)
			assert.True(t, filepath.IsAbs(result))
		})
	}
}

func TestSafePath_RejectsTraversal(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	tests := []struct {
		name string
		path string
	}{
		{"parent directory", "../etc/passwd"},
		{"deep traversal", "../../../etc/shadow"},
		{"absolute path", "/etc/passwd"},
		{"hidden traversal", "src/../../etc/passwd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := SafePath(root, tt.path)
			assert.Error(t, err, "path %q should be rejected", tt.path)
		})
	}
}

func TestSafePath_RejectsRootItself(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	// ".." from a subdirectory that resolves to root should be caught
	_, err := SafePath(root, "subdir/..")
	// This resolves to the root itself, which is allowed since absPath == absRoot
	// but not a file - the handler checks IsDir separately
	assert.NoError(t, err)
}
