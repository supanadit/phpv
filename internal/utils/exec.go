package utils

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
)

type ExecContext struct {
	Stdout  io.Writer
	Stderr  io.Writer
	Filter  *ErrorWarningFilter
	Verbose bool
}

func NewExecContext(verbose bool) *ExecContext {
	ctx := &ExecContext{
		Verbose: verbose,
	}
	if verbose {
		ctx.Stdout = os.Stdout
		ctx.Stderr = os.Stderr
	} else {
		ctx.Stdout = io.Discard
		ctx.Filter = NewErrorWarningFilter(os.Stderr)
		ctx.Stderr = ctx.Filter
	}
	return ctx
}

func (ctx *ExecContext) Command(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.Stdout = ctx.Stdout
	cmd.Stderr = ctx.Stderr
	return cmd
}

func (ctx *ExecContext) Run(cmd *exec.Cmd) error {
	if ctx.Verbose {
		fmt.Println("Running", cmd.Args[0], "with args:", cmd.Args[1:])
	}
	if err := cmd.Run(); err != nil {
		ctx.Flush()
		return err
	}
	ctx.Flush()
	return nil
}

func (ctx *ExecContext) Flush() {
	if ctx.Filter != nil {
		ctx.Filter.Flush()
	}
}

func GetJobs(jobs int) int {
	if jobs == 0 {
		return runtime.NumCPU()
	}
	return jobs
}
