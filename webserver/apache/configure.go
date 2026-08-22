package apache

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/webserver"
)

// Configure wires a PHP version into Apache using the selected connector mode.
func (s *ApacheService) Configure(opts webserver.ConfigureOptions) error {
	installed, version := s.IsInstalled()
	if !installed {
		return fmt.Errorf("apache is not installed. Run `phpv apache install <version>` first")
	}
	if opts.PHPVersion == "" {
		return fmt.Errorf("a PHP version is required (use --php) or install/set a default with phpv")
	}
	connector := opts.Connector
	if connector == "" {
		connector = domain.ConnectorFPM
	}
	if !connector.IsValid() {
		return fmt.Errorf("unsupported connector: %q (use fpm, cgi, or mod_php)", connector)
	}

	prefix := s.httpdPrefix(version)

	if err := s.ensureBinary(opts.PHPVersion, connector); err != nil {
		return err
	}

	cfg := &domain.WebserverConfig{
		ServerType:  "apache",
		Version:     version,
		Prefix:      prefix,
		PHPVersion:  opts.PHPVersion,
		Connector:   string(connector),
		MPM:         defaultMPM(connector, opts.MPM),
		ListenPort:  opts.Port,
		User:        opts.User,
		Group:       opts.Group,
		FPMBasePort: 9000,
		Vhosts:      []domain.Vhost{},
	}
	if cfg.ListenPort == 0 {
		cfg.ListenPort = 8080
	}
	if cfg.User == "" {
		cfg.User = os.Getenv("USER")
	}
	if cfg.Group == "" {
		cfg.Group = primaryGroup()
	}

	if err := saveConfig(cfg); err != nil {
		return err
	}
	if err := s.writeBaseConfig(cfg, prefix); err != nil {
		return err
	}
	if connector == domain.ConnectorFPM {
		if err := s.writeFPMPool(cfg); err != nil {
			return err
		}
	}
	if err := s.applyMPM(cfg, prefix); err != nil {
		return err
	}
	if err := s.applyServerIdentity(cfg, prefix); err != nil {
		return err
	}

	fmt.Printf("✓ Apache configured: PHP %s via %s (MPM %s, port %d)\n",
		cfg.PHPVersion, cfg.Connector, cfg.MPM, cfg.ListenPort)
	fmt.Println("Run `phpv apache start` to launch Apache.")
	return nil
}

// ensureBinary verifies the PHP build has the binary required by the chosen
// connector; it prompts to rebuild if missing.
func (s *ApacheService) ensureBinary(phpVersion string, connector domain.ConnectorMode) error {
	phpBin := filepath.Join(s.siloSvc.PackagePrefix("php", phpVersion), "bin", "php")
	if _, err := os.Stat(phpBin); err != nil {
		return fmt.Errorf("PHP %s is not installed. Run `phpv install %s` first", phpVersion, phpVersion)
	}

	var binary string
	switch connector {
	case domain.ConnectorFPM:
		binary = "php-fpm"
	case domain.ConnectorCGI:
		binary = "php-cgi"
	case domain.ConnectorModPHP:
		// libphp.so — checked during writeBaseConfig
		return nil
	default:
		return nil
	}
	binaryPath := filepath.Join(filepath.Dir(phpBin), binary)
	if _, err := os.Stat(binaryPath); err == nil {
		return nil
	}
	fmt.Printf("PHP %s was not built with %s support (no %s binary).\n", phpVersion, connector, binary)
	fmt.Println("Rebuild PHP with the required flag? [Y/n] ")
	answer := readLine()
	if answer == "n" || answer == "no" {
		return fmt.Errorf("%s requires PHP built with %s support; rebuild with `phpv install %s --%s --fresh`",
			connector, binary, phpVersion, connector)
	}
	return s.rebuild(phpVersion, connector)
}

// rebuild re-runs the PHP build with the requested connector flag, preserving
// the existing extensions.
func (s *ApacheService) rebuild(phpVersion string, connector domain.ConnectorMode) error {
	manifest, err := s.siloSvc.GetExtensionManifest(phpVersion)
	extensions := []string{}
	if err == nil && manifest != nil {
		for _, e := range manifest.Extensions {
			extensions = append(extensions, e.Name)
		}
	}
	if len(extensions) == 0 {
		defaults, _ := s.assemblerSvc.Graph().DefaultExtensions(phpVersion)
		extensions = defaults
	}
	fmt.Printf("Rebuilding PHP %s with connector %s (this may take a while)...\n", phpVersion, connector)
	if _, err := s.assemblerSvc.Assemble(s.ctx, "php", phpVersion, false, extensions, true, nil, nil, 0, true, connector); err != nil {
		return fmt.Errorf("rebuild php: %w", err)
	}
	return nil
}
