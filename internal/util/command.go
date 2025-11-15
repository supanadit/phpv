package util

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// RunCommand runs a command in the specified directory and streams output to stdout/stderr
func RunCommand(ctx context.Context, dir string, env []string, name string, args ...string) error {
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
