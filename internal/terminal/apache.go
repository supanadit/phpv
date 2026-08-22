package terminal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/webserver"
)

func (h *PHPHandler) apacheCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apache",
		Short: "Manage Apache HTTP Server",
		Long: `Install, configure, and manage a user-space Apache httpd with PHP integration.

Apache (plus APR, APR-util, PCRE2, expat) is built from source and stored under
the phpv root. phpv manages PHP-FPM/CGI/mod_php integration, virtual hosts, and
the httpd process — all without touching system directories.

Examples:
  phpv apache install 2.4.66            Build and install Apache httpd
  phpv apache configure --php 8.4       Connect PHP 8.4 via FPM (default)
  phpv apache vhost add . test.local    Serve the current directory at test.local
  phpv apache start                     Launch Apache (and php-fpm) in the background`,
	}

	cmd.AddCommand(h.apacheInstallCmd())
	cmd.AddCommand(h.apacheUninstallCmd())
	cmd.AddCommand(h.apacheConfigureCmd())
	cmd.AddCommand(h.apacheStartCmd())
	cmd.AddCommand(h.apacheStopCmd())
	cmd.AddCommand(h.apacheRestartCmd())
	cmd.AddCommand(h.apacheStatusCmd())
	cmd.AddCommand(h.apacheVhostCmd())
	return cmd
}

func (h *PHPHandler) apacheInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install <version>",
		Short: "Build and install Apache httpd",
		Long:  "Build Apache httpd (with APR, APR-util, PCRE2, expat) from source into the phpv root.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return h.apacheSvc.Install(args[0])
		},
	}
}

func (h *PHPHandler) apacheUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Remove Apache httpd",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return h.apacheSvc.Uninstall()
		},
	}
}

func (h *PHPHandler) apacheConfigureCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "configure",
		Short: "Configure PHP integration with Apache",
		Long: `Configure how Apache runs PHP.

The connector selects how Apache talks to PHP:
  fpm      (default) proxy to a php-fpm pool via mod_proxy_fcgi
  cgi      spawn php-cgi via mod_fcgid
  mod_php  load PHP compiled as an Apache module (requires prefork MPM)

Examples:
  phpv apache configure --php 8.4 --connector fpm --mpm event
  phpv apache configure --php 8.4 --connector mod_php --mpm prefork`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			phpVersion, _ := cmd.Flags().GetString("php")
			connectorStr, _ := cmd.Flags().GetString("connector")
			mpm, _ := cmd.Flags().GetString("mpm")
			port, _ := cmd.Flags().GetInt("port")

			if phpVersion == "" {
				active, err := h.resolveActiveVersion()
				if err != nil {
					return fmt.Errorf("specify --php (could not resolve an active version: %w)", err)
				}
				phpVersion = active
			} else {
				resolved, err := h.resolveVersion(phpVersion)
				if err == nil {
					phpVersion = resolved
				}
			}

			connector := domain.ConnectorMode(connectorStr)
			if connectorStr == "" {
				connector = domain.ConnectorFPM
			}
			if !connector.IsValid() {
				return fmt.Errorf("invalid connector %q (use fpm, cgi, or mod_php)", connectorStr)
			}
			return h.apacheSvc.Configure(webserver.ConfigureOptions{
				PHPVersion: phpVersion,
				Connector:  connector,
				MPM:        mpm,
				Port:       port,
			})
		},
	}
	cmd.Flags().String("php", "", "PHP version to connect (default: active version)")
	cmd.Flags().String("connector", "fpm", "Connector mode: fpm, cgi, or mod_php")
	cmd.Flags().String("mpm", "", "MPM: event, worker, or prefork")
	cmd.Flags().Int("port", 0, "Listen port (default 8080)")
	return cmd
}

func (h *PHPHandler) apacheStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start Apache (and PHP-FPM if configured)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			foreground, _ := cmd.Flags().GetBool("foreground")
			return h.apacheSvc.Start(foreground)
		},
	}
	cmd.Flags().Bool("foreground", false, "Run in the foreground (blocking)")
	return cmd
}

func (h *PHPHandler) apacheStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop Apache (and PHP-FPM if configured)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return h.apacheSvc.Stop()
		},
	}
}

func (h *PHPHandler) apacheRestartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: "Restart Apache (and PHP-FPM if configured)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return h.apacheSvc.Restart()
		},
	}
}

func (h *PHPHandler) apacheStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show Apache and PHP-FPM status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := h.apacheSvc.Status()
			if err != nil {
				return err
			}
			fmt.Println(status)
			return nil
		},
	}
}

func (h *PHPHandler) apacheVhostCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vhost",
		Short: "Manage Apache virtual hosts",
		Long: `Manage Apache virtual hosts. Vhosts are stored under the phpv root and
included automatically, so adding/removing them needs no sudo.

Examples:
  phpv apache vhost add . test.local            Serve the current directory
  phpv apache vhost add /srv/app app.local --ssl  (Enable SSL/port 443)
  phpv apache vhost list
  phpv apache vhost remove test.local
  phpv apache vhost disable test.local`,
	}

	addCmd := &cobra.Command{
		Use:   "add <docroot> <domain>",
		Short: "Add a virtual host",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			docroot := args[0]
			if docroot == "." || docroot == "" {
				wd, err := os.Getwd()
				if err != nil {
					return err
				}
				docroot = wd
			} else {
				abs, err := filepath.Abs(docroot)
				if err == nil {
					docroot = abs
				}
			}
			if fi, err := os.Stat(docroot); err != nil || !fi.IsDir() {
				return fmt.Errorf("document root %q is not a directory", docroot)
			}

			ssl, _ := cmd.Flags().GetBool("ssl")
			port, _ := cmd.Flags().GetInt("port")
			alias, _ := cmd.Flags().GetString("alias")
			phpVersion, _ := cmd.Flags().GetString("php-version")
			fpmPort, _ := cmd.Flags().GetInt("fpm-port")

			vhost := domain.Vhost{
				ServerName:   args[1],
				DocumentRoot: docroot,
				Port:         port,
				SSLEnabled:   ssl,
				PHPVersion:   phpVersion,
				FPMPort:      fpmPort,
				Enabled:      true,
			}
			if alias != "" {
				vhost.Aliases = strings.Fields(alias)
			}
			return h.apacheSvc.VhostAdd(vhost)
		},
	}
	addCmd.Flags().Bool("ssl", false, "Enable SSL")
	addCmd.Flags().Int("port", 0, "Listen port (default: Apache's configured port)")
	addCmd.Flags().String("alias", "", "Space-separated ServerAlias list")
	addCmd.Flags().String("php-version", "", "Per-vhost PHP version (FPM mode only)")
	addCmd.Flags().Int("fpm-port", 0, "Per-vhost FPM backend port (FPM mode only)")
	cmd.AddCommand(addCmd)

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List virtual hosts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			vhosts, err := h.apacheSvc.VhostList()
			if err != nil {
				return err
			}
			if len(vhosts) == 0 {
				fmt.Println("No virtual hosts configured.")
				return nil
			}
			for _, v := range vhosts {
				state := "enabled"
				if !v.Enabled {
					state = "disabled"
				}
				port := v.Port
				if port == 0 {
					port = 80
				}
				fmt.Printf("  %s  %s:%d  ->  %s  [%s]\n", v.ServerName, v.ServerName, port, v.DocumentRoot, state)
			}
			return nil
		},
	}
	cmd.AddCommand(listCmd)

	removeCmd := &cobra.Command{
		Use:   "remove <domain>",
		Short: "Remove a virtual host",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return h.apacheSvc.VhostRemove(args[0])
		},
	}
	cmd.AddCommand(removeCmd)

	enableCmd := &cobra.Command{
		Use:   "enable <domain>",
		Short: "Enable a virtual host",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return h.apacheSvc.VhostEnable(args[0])
		},
	}
	cmd.AddCommand(enableCmd)

	disableCmd := &cobra.Command{
		Use:   "disable <domain>",
		Short: "Disable a virtual host",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return h.apacheSvc.VhostDisable(args[0])
		},
	}
	cmd.AddCommand(disableCmd)

	return cmd
}
