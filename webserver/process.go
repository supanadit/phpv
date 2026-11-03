package webserver

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// SignalTerm sends SIGTERM to a pid, ignoring already-dead targets.
func SignalTerm(pid int) error {
	if pid <= 0 {
		return nil
	}
	err := syscall.Kill(pid, syscall.SIGTERM)
	if err != nil && err != syscall.ESRCH {
		return err
	}
	return nil
}

// AwaitExit waits up to grace for the pid to disappear, then SIGKILLs it.
func AwaitExit(pid int, grace time.Duration) {
	if pid <= 0 {
		return
	}
	deadline := time.Now().Add(grace)
	for time.Now().Before(deadline) {
		if !ProcessAlive(pid) {
			return
		}
		time.Sleep(150 * time.Millisecond)
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
}

// Detach starts cmd in its own session so it survives the phpv process,
// redirecting stdout/stderr to the given log file. Returns the child pid.
func Detach(cmd *exec.Cmd, logPath string) (int, error) {
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return 0, fmt.Errorf("open log %s: %w", logPath, err)
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		logFile.Close()
		return 0, err
	}
	pid := cmd.Process.Pid
	// Release our handle: the daemon outlives this phpv invocation.
	go func() { _ = cmd.Wait() }()
	return pid, nil
}
