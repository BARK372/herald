package notify

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockNotifier struct {
	mu     sync.Mutex
	events []Event
}

func (m *mockNotifier) Notify(e Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, e)
}

func (m *mockNotifier) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.events)
}

func TestHub_BroadcastsToAllNotifiers(t *testing.T) {
	t.Parallel()

	n1 := &mockNotifier{}
	n2 := &mockNotifier{}
	hub := NewHub(n1, n2)

	hub.Notify(Event{Type: "task.completed", TaskID: "t1"})

	// Hub dispatches asynchronously
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 1, n1.count())
	assert.Equal(t, 1, n2.count())
}
