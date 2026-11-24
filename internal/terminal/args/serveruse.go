package args

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	webidentity "github.com/supanadit/phpv/pkg/webidentity"
)

func (h *PHPHandler) serveruse() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "use <caddy|nginx|apache>",
		Short:             "Associate an installed webserver with a PHP version",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: serverNameCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			server := strings.ToLower(args[0])
			if !webidentity.IsValidServerName(server) {
				return invalidServerName(server)
			}
			if !containsString(h.installedServers(), server) {
				return fmt.Errorf("%s is not installed — run: phpv server install %s", server, server)
			}
			phpFlag, _ := cmd.Flags().GetString("php")
			phpVersion, err := h.resolveVersion(phpFlag)
			if err != nil {
				return err
			}
			sapi, err := resolveCommandSAPI(cmd, server)
			if err != nil {
				return err
			}
			if err := h.recordServerSelection(phpVersion, server, sapi); err != nil {
				return fmt.Errorf("associate %s with PHP %s: %w", server, phpVersion, err)
			}
			fmt.Printf("✓ %s (%s) is now associated with PHP %s\n", server, sapiLabel(sapi), phpVersion)
			return nil
		},
	}
	cmd.Flags().String("php", "", "PHP version to associate with (default: active version)")
	cmd.Flags().String("sapi", "", "PHP SAPI: fpm (default) or modphp (apache only)")
	_ = cmd.RegisterFlagCompletionFunc("php", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return h.versionNameCompletion(toComplete)
	})
	_ = cmd.RegisterFlagCompletionFunc("sapi", sapiCompletion)

	return cmd
}
