package executor

import (
	"fmt"
	"sort"
	"sync"
)

// Factory creates an Executor from a configuration map.
type Factory func(cfg map[string]any) (Executor, error)

var (
	registryMu sync.RWMutex
	registry   = map[string]Factory{}
)

// Register adds an executor factory under the given name.
// Typically called from init() in each executor implementation.
func Register(name string, factory Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()

	if _, dup := registry[name]; dup {
		panic(fmt.Sprintf("executor: duplicate registration for %q", name))
	}
	registry[name] = factory
}

// Get returns the factory for the named executor, if registered.
func Get(name string) (Factory, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	f, ok := registry[name]
	return f, ok
}

// Available returns a sorted list of registered executor names.
func Available() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
