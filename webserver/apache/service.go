package apache

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/supanadit/phpv/assembler"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/appctx"
	"github.com/supanadit/phpv/silo"
	"github.com/supanadit/phpv/system"
)

// ApacheService implements webserver.WebServer for Apache httpd. It installs
// Apache (plus APR/APR-util/expat/PCRE2) from source under the phpv root, then
// manages PHP-FPM integration, vhosts, and the httpd process — all in user
// space with no system dirs touched.
type ApacheService struct {
	ctx          context.Context
	siloSvc      *silo.Service
	assemblerSvc *assembler.Service
	systemSvc    *system.Service
}

// NewService creates an ApacheService.
func NewService(ac appctx.AppContext, siloSvc *silo.Service, assemblerSvc *assembler.Service, systemSvc *system.Service) *ApacheService {
	return &ApacheService{
		ctx:          ac.Ctx,
		siloSvc:      siloSvc,
		assemblerSvc: assemblerSvc,
		systemSvc:    systemSvc,
	}
}

// Name returns the webserver identifier.
func (s *ApacheService) Name() string { return "apache" }

// httpdPrefix returns the install prefix for an httpd version.
func (s *ApacheService) httpdPrefix(version string) string {
	return s.siloSvc.PackagePrefix("httpd", version)
}

// IsInstalled reports whether httpd is installed and its version.
func (s *ApacheService) IsInstalled() (bool, string) {
	cfg, err := loadConfig()
	if err != nil || cfg == nil || cfg.Version == "" {
		return false, ""
	}
	apacheBin := filepath.Join(s.httpdPrefix(cfg.Version), "bin", "httpd")
	if _, err := os.Stat(apacheBin); err != nil {
		return false, cfg.Version
	}
	return true, cfg.Version
}

// Install builds Apache httpd (plus APR/APR-util/expat/PCRE2) from source.
func (s *ApacheService) Install(version string) error {
	fmt.Printf("Installing Apache httpd %s (with APR, APR-util, PCRE2, expat)...\n", version)

	if err := s.checkBuildDeps(); err != nil {
		return err
	}

	if _, err := s.assemblerSvc.Assemble(s.ctx, "httpd", version, false, nil, true, nil, nil, 0, false, domain.ConnectorNone); err != nil {
		return fmt.Errorf("install httpd: %w", err)
	}

	exactVersion, err := s.assemblerSvc.ResolveVersion("httpd", version)
	if err != nil {
		return fmt.Errorf("resolve httpd version: %w", err)
	}
	prefix := s.httpdPrefix(exactVersion)

	if err := s.postInstall(exactVersion, prefix); err != nil {
		return err
	}
	fmt.Printf("✓ Apache httpd %s installed at %s\n", exactVersion, prefix)
	return nil
}

// Uninstall removes httpd (and its dependency packages) from the store.
func (s *ApacheService) Uninstall() error {
	installed, version := s.IsInstalled()
	if !installed {
		return fmt.Errorf("apache is not installed")
	}
	prefix := s.httpdPrefix(version)
	fmt.Printf("Removing %s...\n", prefix)
	if err := os.RemoveAll(prefix); err != nil {
		return err
	}
	if err := os.RemoveAll(apacheRoot()); err != nil {
		return err
	}
	fmt.Println("✓ Apache uninstalled")
	return nil
}
