// Package args owns the terminal channel's root command plumbing.
package args

import (
	"github.com/spf13/cobra"

	"github.com/supanadit/phpv/domain"
)

// NewRootCmd builds the root command shared by every terminal command.
func NewRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:   domain.AppName,
		Short: "PHP version manager",
		// Runtime failures are not usage mistakes: print the error once
		// (main does that) instead of dumping the flag reference too.
		SilenceUsage:  true,
		SilenceErrors: true,
	}
}
