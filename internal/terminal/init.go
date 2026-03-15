package terminal

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/pflag"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/shell"
)

type ShellService interface {
	Init(shellType string) (string, error)
	Use(ctx context.Context, versionSpec string) (string, error)
	Unuse(ctx context.Context) string
	SetDefault(ctx context.Context, versionSpec string) error
	GetDefault(ctx context.Context) (*domain.Version, error)
	GetCurrent(ctx context.Context) (*domain.Version, error)
	Which(ctx context.Context) (string, error)
	ListInstalled(ctx context.Context) ([]domain.Version, error)
}

type ShellHandler struct {
	service *shell.Service
}

func NewShellHandler(ctx context.Context, svc *shell.Service) bool {
	handler := &ShellHandler{
		service: svc,
	}

	pflag.Parse()
	args := pflag.Args()

	if len(args) == 0 {
		return false
	}

	switch args[0] {
	case "init":
		return handler.handleInit(ctx, args)
	case "use":
		return handler.handleUse(ctx, args)
	case "shell":
		return handler.handleShell(ctx, args)
	case "default":
		return handler.handleDefault(ctx, args)
	case "versions":
		return handler.handleVersions(ctx, args)
	case "which":
		return handler.handleWhich(ctx, args)
	}

	return false
}

func (h *ShellHandler) handleInit(ctx context.Context, args []string) bool {
	shellType := ""
	if len(args) > 1 {
		shellType = args[1]
	}

	output, err := h.service.Init(shellType)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return true
	}

	fmt.Print(output)
	return true
}

func (h *ShellHandler) handleUse(ctx context.Context, args []string) bool {
	if len(args) < 2 {
		fmt.Println("Usage: phpv use <version>")
		fmt.Println("Example: phpv use 8.3")
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

func (h *ShellHandler) handleShell(ctx context.Context, args []string) bool {
	if len(args) < 2 {
		fmt.Println("Usage: phpv shell <version>")
		fmt.Println("Example: phpv shell 8.3")
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

func (h *ShellHandler) handleDefault(ctx context.Context, args []string) bool {
	if len(args) < 2 {
		current, err := h.service.GetDefault(ctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			return true
		}
		if current != nil {
			fmt.Printf("Default PHP version is %s\n", current.String())
		} else {
			fmt.Println("No default PHP version set")
		}
		return true
	}

	err := h.service.SetDefault(ctx, args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return true
	}

	fmt.Printf("Default PHP version set to %s\n", args[1])
	return true
}

func (h *ShellHandler) handleVersions(ctx context.Context, args []string) bool {
	versions, err := h.service.ListInstalled(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return true
	}

	if len(versions) == 0 {
		fmt.Println("No PHP versions installed")
		return true
	}

	current, _ := h.service.GetCurrent(ctx)
	defaultVer, _ := h.service.GetDefault(ctx)

	for _, v := range versions {
		marker := " "
		if current != nil && v.String() == current.String() {
			marker = "*"
		} else if defaultVer != nil && v.String() == defaultVer.String() {
			marker = ">"
		}
		fmt.Printf("%s %s\n", marker, v.String())
	}

	return true
}

func (h *ShellHandler) handleWhich(ctx context.Context, args []string) bool {
	path, err := h.service.Which(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return true
	}

	fmt.Println(path)
	return true
}
