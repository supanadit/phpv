package terminal

import (
	"github.com/spf13/cobra"
	"github.com/supanadit/phpv/domain"
	"go.uber.org/fx"
)

func ExecuteCobra(handler *TerminalHandler, shutdowner fx.Shutdowner) error {
	rootCmd := &cobra.Command{
		Use:   "phpv",
		Short: "PHP Version Manager",
		Long:  `A PHP Version Manager for building and managing multiple PHP versions from source.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	rootCmd.Version = domain.AppVersion

	registerInstallCommands(rootCmd, handler)
	registerVersionCommands(rootCmd, handler)
	registerToolsCommands(rootCmd, handler)
	registerExtraCommands(rootCmd, handler)
	registerPeclCommands(rootCmd, handler)
	registerPharCommands(rootCmd, handler)

	if err := rootCmd.Execute(); err != nil {
		shutdowner.Shutdown(fx.ExitCode(1))
		return err
	}

	return nil
}
