package terminal

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/pflag"
	"github.com/supanadit/phpv/shell"
)

type ShellCommandHandler struct {
	service *shell.Service
}

func NewShellCommandHandler(ctx context.Context, svc *shell.Service) bool {
	handler := &ShellCommandHandler{
		service: svc,
	}

	pflag.Parse()
	args := pflag.Args()

	if len(args) == 0 {
		return false
	}

	switch args[0] {
	case "sh-use":
		return handler.handleShUse(ctx, args)
	case "sh-shell":
		return handler.handleShShell(ctx, args)
	}

	return false
}

func (h *ShellCommandHandler) handleShUse(ctx context.Context, args []string) bool {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: phpv sh-use <version>")
		return true
	}

	output, err := h.service.Use(ctx, args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return true
	}

	fmt.Println(output)
	return true
}

func (h *ShellCommandHandler) handleShShell(ctx context.Context, args []string) bool {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: phpv sh-shell <version>")
		return true
	}

	output, err := h.service.Use(ctx, args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return true
	}

	fmt.Println(output)
	return true
}
