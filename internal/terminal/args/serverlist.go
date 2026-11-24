package args

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/supanadit/phpv/domain"
	webidentity "github.com/supanadit/phpv/pkg/webidentity"
)

func (h *PHPHandler) serverlist() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed webservers, defaults, associations and running instances",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root := resolvePHPVRootForServer()
			fmt.Println("Installed webservers:")
			any := false
			for _, name := range webidentity.ServerNames() {
				pkgDir := filepath.Join(root, "packages", name)
				entries, err := os.ReadDir(pkgDir)
				if err != nil {
					continue
				}
				var versions []string
				for _, e := range entries {
					if e.IsDir() {
						versions = append(versions, e.Name())
					}
				}
				if len(versions) == 0 {
					continue
				}
				any = true
				fmt.Printf("  %-8s %s\n", name, strings.Join(versions, ", "))
			}
			if !any {
				fmt.Println("  (none — try: phpv server install caddy)")
			}

			activePHP, _ := h.resolveVersion("")
			fmt.Println("\nGlobal default:")
			switch def := strings.TrimSpace(os.Getenv(defaultServerEnv)); {
			case def != "":
				envSAPI := webidentity.NormalizeSAPI(os.Getenv(defaultSAPIEnv))
				if strings.TrimSpace(os.Getenv(defaultSAPIEnv)) == "" {
					cfgSAPI, _ := h.configSvc.Get("server_sapi")
					envSAPI = webidentity.NormalizeSAPI(cfgSAPI)
				}
				if envSAPI == "" || (envSAPI == "modphp" && def != domain.ServerApache) {
					envSAPI = "fpm"
				}
				fmt.Printf("  %s (%s, from %s)\n", def, sapiLabel(envSAPI), defaultServerEnv)
			default:
				if cfg, _ := h.configSvc.Get("server"); cfg != "" {
					sapi, _ := h.configSvc.Get("server_sapi")
					if env := strings.TrimSpace(os.Getenv(defaultSAPIEnv)); env != "" {
						sapi = env
					}
					sapi = webidentity.NormalizeSAPI(sapi)
					if sapi == "" || (sapi == "modphp" && cfg != domain.ServerApache) {
						sapi = "fpm"
					}
					fmt.Printf("  %s (%s, from phpv server default)\n", cfg, sapiLabel(sapi))
				} else {
					fmt.Println("  (none — falls back to the first installed server)")
				}
			}

			fmt.Println("\nPer-PHP associations:")
			prefs := loadServerPreferences(root)
			if len(prefs.Servers) == 0 {
				fmt.Println("  (none — run: phpv server use <name>)")
			} else {
				versions := make([]string, 0, len(prefs.Servers))
				for version := range prefs.Servers {
					versions = append(versions, version)
				}
				sort.Strings(versions)
				for _, version := range versions {
					marker := ""
					if version == activePHP {
						marker = "  (active)"
					}
					sapi := "fpm"
					if recorded, ok := prefs.getSAPI(version); ok {
						sapi = webidentity.NormalizeSAPI(recorded)
					}
					fmt.Printf("  PHP %-10s -> %s (%s)%s\n", version, prefs.Servers[version], sapiLabel(sapi), marker)
				}
			}
			if activePHP != "" {
				if server, source, err := h.resolveServerForPHP("", activePHP); err == nil {
					sapi, sapiErr := h.resolveSAPIForPHP("", server, activePHP)
					if sapiErr != nil {
						sapi = "fpm"
					}
					fmt.Printf("\nActive PHP %s -> %s (%s) (%s)\n", activePHP, server, sapiLabel(sapi), serverSourceLabel(source))
				}
			}

			m, err := h.serverManager()
			if err != nil {
				return err
			}
			if _, err := m.Prune(); err != nil {
				return err
			}
			instances, err := m.List()
			if err != nil {
				return err
			}
			fmt.Println("\nInstances:")
			if len(instances) == 0 {
				fmt.Println("  (none)")
				return nil
			}
			for _, inst := range instances {
				state := "stopped"
				if m.Store().Running(inst) {
					state = "running"
				}
				fmt.Printf("  %-28s %-8s %s (%s)  php-%s  %s  %s\n", inst.ID, state, inst.Server, sapiLabel(inst.SAPI), inst.PHPVersion, webidentity.InstanceURL(inst.Host, inst.Port), inst.Dir)
			}
			return nil
		},
	}

	return cmd
}
