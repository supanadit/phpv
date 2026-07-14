package shutdown

import (
	"os"
	"syscall"
	"testing"
)

func TestSignalExitCode(t *testing.T) {
	tests := []struct {
		sig  os.Signal
		code int
	}{
		{syscall.SIGINT, 130},
		{syscall.SIGTERM, 143},
		{syscall.SIGHUP, 129},
		{syscall.SIGUSR1, 1},
	}
	for _, tt := range tests {
		got := SignalExitCode(tt.sig)
		if got != tt.code {
			t.Errorf("SignalExitCode(%v) = %d, want %d", tt.sig, got, tt.code)
		}
	}
}

func TestManagerContextCancelledOnSignal(t *testing.T) {
	m := New(syscall.SIGUSR1)
	defer m.Stop()

	syscall.Kill(syscall.Getpid(), syscall.SIGUSR1)
	sig := m.Wait()

	if sig != syscall.SIGUSR1 {
		t.Fatalf("Wait() = %v, want SIGUSR1", sig)
	}
	if m.ctx.Err() == nil {
		t.Fatal("context should be cancelled after signal")
	}
}

func TestManagerStopNoSignal(t *testing.T) {
	m := New(syscall.SIGUSR1)
	m.Stop()

	if m.ctx.Err() == nil {
		t.Fatal("context should be cancelled after Stop")
	}
}
