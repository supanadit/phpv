package util

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/supanadit/phpv/internal/ui"
)

// RunCommandWithMode runs a command based on the UI output mode
// - Quiet mode: No output except errors
// - Animation mode: Shows spinner, captures output, shows on failure
// - Verbose mode: Streams all output to stdout/stderr
func RunCommandWithMode(ctx context.Context, dir string, env []string, name string, args ...string) error {
	u := ui.GetUI()
	mode := u.OutputMode()

	desc := name
	if len(args) > 0 {
		desc = name + " " + strings.Join(args, " ")
	}

	switch mode {
	case ui.ModeQuiet:
		return RunCommandQuiet(ctx, dir, env, name, args...)
	case ui.ModeAnimation:
		return RunCommandWithAnimation(ctx, dir, env, desc, name, args...)
	case ui.ModeVerbose:
		return RunCommandVerbose(ctx, dir, env, name, args...)
	default:
		return RunCommandVerbose(ctx, dir, env, name, args...)
	}
}

// RunCommandQuiet runs a command with minimal output (only errors)
func RunCommandQuiet(ctx context.Context, dir string, env []string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir

	if env != nil {
		cmd.Env = env
	}

	_, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed: %w", err)
	}
	return nil
}

// RunCommandVerbose runs a command and streams all output to stdout/stderr
func RunCommandVerbose(ctx context.Context, dir string, env []string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if env != nil {
		cmd.Env = env
	}

	fmt.Printf("→ Running: %s %s\n", name, strings.Join(args, " "))
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

// RunCommandWithAnimation runs a command with spinner animation and captures output
// On failure, it displays the captured output
func RunCommandWithAnimation(ctx context.Context, dir string, env []string, description string, name string, args ...string) error {
	u := ui.GetUI()

	u.StartSpinnerWithDisplay(description)
	defer u.StopSpinnerWithClear()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir

	if env != nil {
		cmd.Env = env
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	u.StopSpinnerWithClear()

	if err != nil {
		if stdout.Len() > 0 {
			fmt.Println(string(stdout.Bytes()))
		}
		if stderr.Len() > 0 {
			fmt.Println(string(stderr.Bytes()))
		}
		return fmt.Errorf("%s failed: %w", description, err)
	}

	u.PrintSuccess(fmt.Sprintf("%s done", description))
	return nil
}

// RunCommand is the original function - now defaults to RunCommandWithMode
func RunCommand(ctx context.Context, dir string, env []string, name string, args ...string) error {
	return RunCommandWithMode(ctx, dir, env, name, args...)
}
