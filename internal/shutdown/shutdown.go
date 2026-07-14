package shutdown

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var DefaultSignals = []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP}

type Manager struct {
	ctx    context.Context
	cancel context.CancelFunc
	once   sync.Once
	done   chan os.Signal
}

func New(sig ...os.Signal) *Manager {
	if len(sig) == 0 {
		sig = DefaultSignals
	}
	ctx, cancel := context.WithCancel(context.Background())
	m := &Manager{
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan os.Signal, 1),
	}
	signal.Notify(m.done, sig...)
	return m
}

func (m *Manager) Context() context.Context {
	return m.ctx
}

func (m *Manager) Wait() os.Signal {
	sig := <-m.done
	m.once.Do(func() {
		m.cancel()
	})
	return sig
}

func (m *Manager) Stop() {
	signal.Stop(m.done)
	m.once.Do(func() {
		m.cancel()
	})
}

func SignalExitCode(sig os.Signal) int {
	switch sig {
	case syscall.SIGINT:
		return 130
	case syscall.SIGTERM:
		return 143
	case syscall.SIGHUP:
		return 129
	default:
		return 1
	}
}

func PrintInterrupted(sig os.Signal) {
	fmt.Fprintf(os.Stderr, "\nReceived %s. Shutting down gracefully... (press Ctrl+C again to force)\n", sig)
}
