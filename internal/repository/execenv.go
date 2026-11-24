package repository

import (
	"context"
	"os"
	"os/exec"
	"syscall"
)

func RunCmd(ctx context.Context, cmd *exec.Cmd, verbose bool) ([]byte, error) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			return nil, err
		}
		return nil, nil
	}
	return cmd.CombinedOutput()
}

func MergeEnv(extraEnv []string) []string {
	if len(extraEnv) == 0 {
		return nil
	}
	env := os.Environ()
	for _, e := range extraEnv {
		env = append(env, e)
	}
	return env
}
