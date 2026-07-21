package terminal

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

type pharDef struct {
	Name string
	URL  string
}

var knownPhars = []pharDef{
	{Name: "composer", URL: "https://getcomposer.org/download/latest-stable/composer.phar"},
	{Name: "wp", URL: "https://raw.githubusercontent.com/wp-cli/builds/gh-pages/phar/wp-cli.phar"},
	{Name: "pie", URL: "https://github.com/php/pie/releases/latest/download/pie.phar"},
	{Name: "phpunit", URL: "https://phar.phpunit.de/phpunit.phar"},
}

func (h *PHPHandler) pharCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "phar",
		Short: "Manage PHAR tools",
		Long:  "Install, list, and manage PHAR tools (composer, wp-cli, pie, phpunit).",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "install <name> [version]",
		Short: "Install a PHAR tool",
		Long:  "Download and install a PHAR tool for the active PHP version.",
		Args:  cobra.MinimumNArgs(1),
		RunE:  h.pharInstall,
	})
	listCmd := &cobra.Command{
		Use:   "list [version]",
		Short: "List installed PHAR tools",
		Args:  cobra.MaximumNArgs(1),
		RunE:  h.pharList,
	}
	listCmd.Flags().Bool("json", false, "Output in JSON format")
	cmd.AddCommand(listCmd)
	cmd.AddCommand(&cobra.Command{
		Use:   "which <name>",
		Short: "Show path to a PHAR tool",
		Args:  cobra.ExactArgs(1),
		RunE:  h.pharWhich,
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "update <name> [version]",
		Short: "Update a PHAR tool",
		Long:  "Re-download and update a PHAR tool for the active PHP version.",
		Args:  cobra.MinimumNArgs(1),
		RunE:  h.pharUpdate,
	})
	return cmd
}

func (h *PHPHandler) pharInstall(cmd *cobra.Command, args []string) error {
	name := args[0]

	version, err := h.resolveVersion("")
	if err != nil {
		return err
	}

	var def *pharDef
	for _, p := range knownPhars {
		if p.Name == name {
			def = &p
			break
		}
	}
	if def == nil {
		return fmt.Errorf("unknown phar tool: %s (known: composer, wp, pie, phpunit)", name)
	}

	prefix := h.siloSvc.PackagePrefix("php", version)
	pharDir := filepath.Join(prefix, "phar")
	if err := os.MkdirAll(pharDir, 0755); err != nil {
		return fmt.Errorf("create phar dir: %w", err)
	}

	pharPath := filepath.Join(pharDir, name+".phar")
	fmt.Printf("Downloading %s...\n", def.URL)
	if err := downloadFile(def.URL, pharPath); err != nil {
		return fmt.Errorf("download %s: %w", name, err)
	}

	pharRel := "phar/" + name + ".phar"
	if err := h.shimSvc.WritePhar(name, pharRel); err != nil {
		return fmt.Errorf("write phar shim: %w", err)
	}

	fmt.Printf("✓ %s installed for PHP %s\n", name, version)
	return nil
}

func (h *PHPHandler) pharList(cmd *cobra.Command, args []string) error {
	jsonFlag, _ := cmd.Flags().GetBool("json")

	version := ""
	if len(args) == 1 {
		version = args[0]
	} else {
		var err error
		version, err = h.resolveVersion("")
		if err != nil {
			return err
		}
	}

	prefix := h.siloSvc.PackagePrefix("php", version)
	pharDir := filepath.Join(prefix, "phar")

	entries, err := os.ReadDir(pharDir)
	if err != nil {
		if os.IsNotExist(err) {
			if jsonFlag {
				type pharListEntry struct {
					Name string `json:"name"`
				}
				type pharListResponse struct {
					PHPVersion string          `json:"php_version"`
					Tools      []pharListEntry `json:"tools"`
				}
				return printJSON(jsonResponse{SchemaVersion: 1, Data: pharListResponse{
					PHPVersion: version,
					Tools:      []pharListEntry{},
				}})
			}
			fmt.Printf("No PHAR tools installed for PHP %s.\n", version)
			return nil
		}
		return fmt.Errorf("read phar dir: %w", err)
	}

	var names []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".phar") {
			names = append(names, strings.TrimSuffix(e.Name(), ".phar"))
		}
	}
	sort.Strings(names)

	if jsonFlag {
		type pharListEntry struct {
			Name string `json:"name"`
		}
		type pharListResponse struct {
			PHPVersion string          `json:"php_version"`
			Tools      []pharListEntry `json:"tools"`
		}
		var tools []pharListEntry
		for _, n := range names {
			tools = append(tools, pharListEntry{Name: n})
		}
		return printJSON(jsonResponse{SchemaVersion: 1, Data: pharListResponse{
			PHPVersion: version,
			Tools:      tools,
		}})
	}

	fmt.Printf("PHAR tools for PHP %s:\n", version)
	for _, n := range names {
		fmt.Printf("  %s\n", n)
	}
	return nil
}

func (h *PHPHandler) pharWhich(cmd *cobra.Command, args []string) error {
	name := args[0]

	version, err := h.resolveActiveVersion()
	if err != nil {
		return err
	}

	prefix := h.siloSvc.PackagePrefix("php", version)
	pharPath := filepath.Join(prefix, "phar", name+".phar")
	if _, err := os.Stat(pharPath); os.IsNotExist(err) {
		return fmt.Errorf("%s not installed for PHP %s. Run `phpv phar install %s` first", name, version, name)
	}
	fmt.Println(pharPath)
	return nil
}

func (h *PHPHandler) pharUpdate(cmd *cobra.Command, args []string) error {
	name := args[0]

	version, err := h.resolveActiveVersion()
	if err != nil {
		return err
	}

	var def *pharDef
	for _, p := range knownPhars {
		if p.Name == name {
			def = &p
			break
		}
	}
	if def == nil {
		return fmt.Errorf("unknown phar tool: %s (known: composer, wp, pie, phpunit)", name)
	}

	prefix := h.siloSvc.PackagePrefix("php", version)
	pharDir := filepath.Join(prefix, "phar")
	if err := os.MkdirAll(pharDir, 0755); err != nil {
		return fmt.Errorf("create phar dir: %w", err)
	}

	pharPath := filepath.Join(pharDir, name+".phar")
	fmt.Printf("Downloading %s...\n", def.URL)
	if err := downloadFile(def.URL, pharPath); err != nil {
		return fmt.Errorf("download %s: %w", name, err)
	}

	pharRel := "phar/" + name + ".phar"
	if err := h.shimSvc.WritePhar(name, pharRel); err != nil {
		return fmt.Errorf("write phar shim: %w", err)
	}

	fmt.Printf("✓ %s updated for PHP %s\n", name, version)
	return nil
}

func downloadFile(url, dest string) error {
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status %s", resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	return err
}
