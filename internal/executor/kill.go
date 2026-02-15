package executor

import (
	"os"
	"syscall"
	"time"
)

// GracefulKill sends SIGTERM to the process group, waits, then SIGKILL if still alive.
// This is a generic POSIX utility usable by any executor implementation.
func GracefulKill(pid int) {
	// Send SIGTERM to process group (negative PID)
	_ = syscall.Kill(-pid, syscall.SIGTERM)

	done := make(chan struct{})
	go func() {
		proc, err := os.FindProcess(pid)
		if err == nil {
			_, _ = proc.Wait()
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		_ = syscall.Kill(-pid, syscall.SIGKILL)
	}
}
