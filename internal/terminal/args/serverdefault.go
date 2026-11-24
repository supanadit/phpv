package args

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	webidentity "github.com/supanadit/phpv/pkg/webidentity"
)

func (h *PHPHandler) serverdefault() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "default <caddy|nginx|apache>",
		Short:             "Set the global default webserver for PHP versions without an association",
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
			sapi, err := resolveCommandSAPI(cmd, server)
			if err != nil {
				return err
			}
			if err := h.configSvc.Set("server", server); err != nil {
				return fmt.Errorf("set global default webserver: %w", err)
			}
			if err := h.configSvc.Set("server_sapi", sapi); err != nil {
				return fmt.Errorf("set global default SAPI: %w", err)
			}
			fmt.Printf("✓ global default webserver: %s (%s)\n", server, sapiLabel(sapi))
			return nil
		},
	}
	cmd.Flags().String("sapi", "", "PHP SAPI: fpm (default) or modphp (apache only)")
	_ = cmd.RegisterFlagCompletionFunc("sapi", sapiCompletion)

	return cmd
}
