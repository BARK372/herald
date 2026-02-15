package executor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubExecutor struct{}

func (s *stubExecutor) Execute(_ context.Context, _ Request, _ ProgressFunc) (*Result, error) {
	return &Result{}, nil
}

func (s *stubExecutor) Capabilities() Capabilities {
	return Capabilities{Name: "stub"}
}

func TestGet_WhenNotRegistered_ReturnsFalse(t *testing.T) {
	t.Parallel()

	_, ok := Get("nonexistent-executor")
	assert.False(t, ok)
}

func TestAvailable_ReturnsSortedNames(t *testing.T) {
	t.Parallel()

	// Available() should return at least the executors registered via init().
	// We can't test exact contents without knowing what's imported,
	// but we can verify it returns a sorted slice.
	names := Available()
	for i := 1; i < len(names); i++ {
		assert.True(t, names[i-1] <= names[i], "Available() should return sorted names")
	}
}

func TestRegister_WhenDuplicate_Panics(t *testing.T) {
	t.Parallel()

	// Register a unique name first
	uniqueName := "test-dup-" + t.Name()
	Register(uniqueName, func(_ map[string]any) (Executor, error) {
		return &stubExecutor{}, nil
	})

	assert.Panics(t, func() {
		Register(uniqueName, func(_ map[string]any) (Executor, error) {
			return &stubExecutor{}, nil
		})
	})
}

func TestRegisterAndGet_RoundTrip(t *testing.T) {
	t.Parallel()

	uniqueName := "test-roundtrip-" + t.Name()
	Register(uniqueName, func(_ map[string]any) (Executor, error) {
		return &stubExecutor{}, nil
	})

	factory, ok := Get(uniqueName)
	require.True(t, ok)

	exec, err := factory(nil)
	require.NoError(t, err)
	assert.Equal(t, "stub", exec.Capabilities().Name)
}
