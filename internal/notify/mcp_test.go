package notify

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSender records all MCP notifications sent.
type mockSender struct {
	mu        sync.Mutex
	targeted  []sentNotification
	broadcast []sentNotification
}

type sentNotification struct {
	sessionID string
	method    string
	params    map[string]any
}

func (m *mockSender) SendNotificationToSpecificClient(sessionID, method string, params map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.targeted = append(m.targeted, sentNotification{sessionID: sessionID, method: method, params: params})
	return nil
}

func (m *mockSender) SendNotificationToAllClients(method string, params map[string]any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.broadcast = append(m.broadcast, sentNotification{method: method, params: params})
}

func (m *mockSender) targetedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.targeted)
}

func (m *mockSender) broadcastCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.broadcast)
}

func (m *mockSender) allTargeted() []sentNotification {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]sentNotification, len(m.targeted))
	copy(cp, m.targeted)
	return cp
}

func (m *mockSender) allBroadcast() []sentNotification {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]sentNotification, len(m.broadcast))
	copy(cp, m.broadcast)
	return cp
}

func TestMCPNotifier_ProgressDebounce(t *testing.T) {
	t.Parallel()

	sender := &mockSender{}
	n := NewMCPNotifier(sender, 100*time.Millisecond)

	// First progress event should go through
	n.Notify(Event{Type: "task.progress", TaskID: "t1", Message: "step 1"})
	assert.Equal(t, 1, sender.broadcastCount())

	// Rapid second event within debounce window should be dropped
	n.Notify(Event{Type: "task.progress", TaskID: "t1", Message: "step 2"})
	assert.Equal(t, 1, sender.broadcastCount())

	// Wait for debounce window to expire
	time.Sleep(120 * time.Millisecond)

	// Third event after debounce should go through
	n.Notify(Event{Type: "task.progress", TaskID: "t1", Message: "step 3"})
	assert.Equal(t, 2, sender.broadcastCount())
}

func TestMCPNotifier_TerminalEventsNoDebounce(t *testing.T) {
	t.Parallel()

	for _, eventType := range []string{"task.completed", "task.failed", "task.cancelled"} {
		sender := &mockSender{}
		n := NewMCPNotifier(sender, 10*time.Second) // large debounce

		// Send a progress first to establish a debounce entry
		n.Notify(Event{Type: "task.progress", TaskID: "t1", Message: "working"})

		// Terminal event must always go through regardless of debounce
		n.Notify(Event{Type: eventType, TaskID: "t1", Message: "done"})

		assert.Equal(t, 2, sender.broadcastCount(), "terminal event %q should not be debounced", eventType)
	}
}

func TestMCPNotifier_StartedAlwaysSent(t *testing.T) {
	t.Parallel()

	sender := &mockSender{}
	n := NewMCPNotifier(sender, 10*time.Second)

	n.Notify(Event{Type: "task.started", TaskID: "t1", Message: "started"})
	assert.Equal(t, 1, sender.broadcastCount())

	msgs := sender.allBroadcast()
	require.Len(t, msgs, 1)
	assert.Equal(t, "notifications/message", msgs[0].method)
}

func TestMCPNotifier_TargetsSpecificSession(t *testing.T) {
	t.Parallel()

	sender := &mockSender{}
	n := NewMCPNotifier(sender, 50*time.Millisecond)

	n.Notify(Event{
		Type:         "task.completed",
		TaskID:       "t1",
		Project:      "myproj",
		Message:      "done",
		MCPSessionID: "sess-abc",
	})

	assert.Equal(t, 1, sender.targetedCount())
	assert.Equal(t, 0, sender.broadcastCount())

	targeted := sender.allTargeted()
	require.Len(t, targeted, 1)
	assert.Equal(t, "sess-abc", targeted[0].sessionID)
	assert.Equal(t, "notifications/message", targeted[0].method)

	data, ok := targeted[0].params["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "t1", data["task_id"])
	assert.Equal(t, "myproj", data["project"])
}

func TestMCPNotifier_BroadcastsWhenNoSession(t *testing.T) {
	t.Parallel()

	sender := &mockSender{}
	n := NewMCPNotifier(sender, 50*time.Millisecond)

	n.Notify(Event{
		Type:    "task.completed",
		TaskID:  "t1",
		Message: "done",
		// No MCPSessionID
	})

	assert.Equal(t, 0, sender.targetedCount())
	assert.Equal(t, 1, sender.broadcastCount())
}

func TestMCPNotifier_ProgressMethodAndParams(t *testing.T) {
	t.Parallel()

	sender := &mockSender{}
	n := NewMCPNotifier(sender, 50*time.Millisecond)

	n.Notify(Event{Type: "task.progress", TaskID: "t1", Message: "compiling..."})

	msgs := sender.allBroadcast()
	require.Len(t, msgs, 1)
	assert.Equal(t, "notifications/progress", msgs[0].method)
	assert.Equal(t, "t1", msgs[0].params["progressToken"])
	assert.Equal(t, "compiling...", msgs[0].params["message"])
}

func TestMCPNotifier_CompletedClearsDebounce(t *testing.T) {
	t.Parallel()

	sender := &mockSender{}
	n := NewMCPNotifier(sender, 10*time.Second) // large debounce

	// Progress sets debounce
	n.Notify(Event{Type: "task.progress", TaskID: "t1", Message: "step 1"})
	assert.Equal(t, 1, sender.broadcastCount())

	// Complete clears debounce
	n.Notify(Event{Type: "task.completed", TaskID: "t1", Message: "done"})

	// Verify debounce was cleared
	n.mu.Lock()
	_, exists := n.lastSent["t1"]
	n.mu.Unlock()
	assert.False(t, exists, "debounce entry should be cleared after completion")
}

func TestMCPNotifier_DifferentTasksIndependentDebounce(t *testing.T) {
	t.Parallel()

	sender := &mockSender{}
	n := NewMCPNotifier(sender, 100*time.Millisecond)

	// t1 progress
	n.Notify(Event{Type: "task.progress", TaskID: "t1", Message: "t1 step 1"})
	// t2 progress — different task, should not be debounced
	n.Notify(Event{Type: "task.progress", TaskID: "t2", Message: "t2 step 1"})

	assert.Equal(t, 2, sender.broadcastCount())

	// t1 again within debounce — should be dropped
	n.Notify(Event{Type: "task.progress", TaskID: "t1", Message: "t1 step 2"})
	assert.Equal(t, 2, sender.broadcastCount())
}
