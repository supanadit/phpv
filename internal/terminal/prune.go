package terminal

import (
	"context"
	"fmt"
	"os"

	"github.com/supanadit/phpv/prune"
)

func NewPruneHandler(ctx context.Context, pruneSvc *prune.Service) bool {
	if len(os.Args) < 2 {
		return false
	}

	if os.Args[1] != "prune" {
		return false
	}

	if err := pruneSvc.Prune(); err != nil {
		panic(err)
	}

	return true
}

func NewCleanSourceHandler(ctx context.Context, pruneSvc *prune.Service) bool {
	if len(os.Args) < 2 {
		return false
	}

	if os.Args[1] != "clean-source" {
		return false
	}

	if len(os.Args) < 3 {
		fmt.Println("Error: Please specify a version to clean")
		fmt.Println("Example:")
		fmt.Println("  phpv clean-source 4.4.9")
		fmt.Println("  phpv clean-source 8.3.0")
		os.Exit(1)
	}

	version := os.Args[2]

	if err := pruneSvc.CleanSource(version); err != nil {
		panic(err)
	}

	return true
}
