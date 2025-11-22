package terminal

import (
	"context"
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
