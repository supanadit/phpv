package bundler

import (
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/afero"
	"github.com/supanadit/phpv/advisor"
	"github.com/supanadit/phpv/assembler"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/download"
	"github.com/supanadit/phpv/forge"
	"github.com/supanadit/phpv/pattern"
	"github.com/supanadit/phpv/source"
	"github.com/supanadit/phpv/unload"
)

type BundlerRepository interface {
	Install(version string) (domain.Forge, error)
	Orchestrate(name, exactVersion string) (domain.Forge, error)
}

type BundlerService struct {
	assemblerSvc    *assembler.AssemblerService
	advisorSvc      *advisor.Service
	forgeSvc        *forge.Service
	downloadSvc     *download.Service
	unloadSvc       *unload.Service
	sourceSvc       *source.Service
	patternRegistry *pattern.PatternRegistry
	silo            *domain.Silo
	fs              afero.Fs
	jobs            int
}

type BundlerServiceConfig struct {
	Assembler assembler.AssemblerRepository
	Advisor   advisor.AdvisorRepository
	Forge     forge.ForgeRepository
	Download  download.DownloadRepository
	Unload    unload.UnloadRepository
	Source    source.SourceRepository
	Silo      *domain.Silo
	Jobs      int
}

func NewBundlerService(cfg BundlerServiceConfig) *BundlerService {
	registry := pattern.NewPatternRegistry()
	registry.RegisterPatterns(pattern.DefaultURLPatterns)

	jobs := cfg.Jobs
	if jobs == 0 {
		jobs = runtime.NumCPU()
	}

	assemblerSvc := assembler.NewAssemblerServiceWithRepo(cfg.Assembler)
	advisorSvc := advisor.NewAdvisorService(cfg.Advisor)
	forgeSvc := forge.NewService(cfg.Forge)
	downloadSvc := download.NewService(cfg.Download)
	unloadSvc := unload.NewService(cfg.Unload)
	sourceSvc := source.NewService(cfg.Source)

	return &BundlerService{
		assemblerSvc:    assemblerSvc,
		advisorSvc:      advisorSvc,
		forgeSvc:        forgeSvc,
		downloadSvc:     downloadSvc,
		unloadSvc:       unloadSvc,
		sourceSvc:       sourceSvc,
		patternRegistry: registry,
		silo:            cfg.Silo,
		fs:              afero.NewOsFs(),
		jobs:            jobs,
	}
}

func (s *BundlerService) Install(version string) (domain.Forge, error) {
	exactVersion, err := s.resolvePHPVersion(version)
	if err != nil {
		return domain.Forge{}, fmt.Errorf("failed to resolve version %q: %w", version, err)
	}
	return s.Orchestrate("php", exactVersion)
}

func (s *BundlerService) resolvePHPVersion(constraint string) (string, error) {
	sources, err := s.sourceSvc.GetVersions()
	if err != nil {
		return "", err
	}

	var phpSources []domain.Source
	for _, src := range sources {
		if src.Name == "php" {
			phpSources = append(phpSources, src)
		}
	}

	parts := strings.Split(constraint, ".")
	major := 0
	minor := 0
	patch := -1

	if len(parts) >= 1 {
		major, _ = strconv.Atoi(parts[0])
	}
	if len(parts) >= 2 {
		minor, _ = strconv.Atoi(parts[1])
	}
	if len(parts) >= 3 {
		patch, _ = strconv.Atoi(parts[2])
	}

	var candidates []domain.Source
	for _, src := range phpSources {
		v := pattern.ParseVersion(src.Version)
		if v.Major != major {
			continue
		}
		if minor > 0 && v.Minor != minor {
			continue
		}
		if patch >= 0 && v.Patch != patch {
			continue
		}
		candidates = append(candidates, src)
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no PHP version found matching %q", constraint)
	}

	sort.Slice(candidates, func(i, j int) bool {
		vi := pattern.ParseVersion(candidates[i].Version)
		vj := pattern.ParseVersion(candidates[j].Version)
		if vi.Major != vj.Major {
			return vi.Major > vj.Major
		}
		if vi.Minor != vj.Minor {
			return vi.Minor > vj.Minor
		}
		return vi.Patch > vj.Patch
	})

	return candidates[0].Version, nil
}

func (s *BundlerService) Orchestrate(name, exactVersion string) (domain.Forge, error) {
	if err := s.ensureBuildTools(); err != nil {
		return domain.Forge{}, fmt.Errorf("failed to ensure build tools: %w", err)
	}

	graph, err := s.assemblerSvc.GetGraph(name, exactVersion)
	if err != nil {
		return domain.Forge{}, fmt.Errorf("failed to resolve dependency graph: %w", err)
	}

	var depOrder []domain.VersionResolved
	processed := make(map[string]bool)

	for _, deps := range graph {
		for _, dep := range deps {
			depVer := extractVersion(dep.Version)
			key := dep.Name + "@" + depVer
			if !processed[key] {
				processed[key] = true
				depOrder = append(depOrder, domain.VersionResolved{
					Package: dep.Name,
					Version: depVer,
				})
			}
		}
	}

	ldLibraryPath := make([]string, 0)
	cppFlags := make([]string, 0)
	ldFlags := make([]string, 0)

	for _, dep := range depOrder {
		if err := s.buildPackage(dep.Package, dep.Version, exactVersion, ldLibraryPath, cppFlags, ldFlags); err != nil {
			return domain.Forge{}, fmt.Errorf("failed to build %s@%s: %w", dep.Package, dep.Version, err)
		}
		depPath := s.silo.DependencyPath(exactVersion, dep.Package, dep.Version)
		ldLibraryPath = append(ldLibraryPath, filepath.Join(depPath, "lib"))
		cppFlags = append(cppFlags, fmt.Sprintf("-I%s/include", depPath))
		ldFlags = append(ldFlags, fmt.Sprintf("-L%s/lib", depPath))
	}

	if err := s.buildPHP(name, exactVersion, ldLibraryPath, cppFlags, ldFlags); err != nil {
		return domain.Forge{}, fmt.Errorf("failed to build PHP: %w", err)
	}

	outputPath := s.silo.PHPOutputPath(exactVersion)
	ldLibraryPath = append(ldLibraryPath, filepath.Join(outputPath, "lib"))

	return domain.Forge{
		Prefix: outputPath,
		Env: map[string]string{
			"LD_LIBRARY_PATH": strings.Join(ldLibraryPath, ":"),
		},
	}, nil
}

func (s *BundlerService) ensureBuildTools() error {
	buildTools := []struct {
		pkg string
		ver string
	}{
		{"m4", "1.4.19"},
		{"autoconf", "2.71"},
		{"automake", "1.17"},
		{"libtool", "2.5.4"},
		{"perl", "5.38.2"},
		{"bison", "1.35"},
		{"flex", "2.5.39"},
	}

	for _, tool := range buildTools {
		toolPath := s.silo.BuildToolBinPath(tool.pkg, tool.ver)

		if exists, _ := afero.DirExists(s.fs, toolPath); exists {
			continue
		}

		if err := s.installBuildTool(tool.pkg, tool.ver); err != nil {
			return fmt.Errorf("failed to install build tool %s@%s: %w", tool.pkg, tool.ver, err)
		}
	}

	return nil
}

func (s *BundlerService) installBuildTool(pkg, version string) error {
	check, err := s.advisorSvc.Check(pkg, version)
	if err != nil {
		return err
	}

	switch check.Action {
	case "skip":
		return nil
	case "download":
		url, err := s.patternRegistry.BuildURLByType(pkg, version, check.SourceType)
		if err != nil {
			return err
		}
		dest := s.silo.GetArchivePath(pkg, version)
		if _, err := s.downloadSvc.Download(url, dest); err != nil {
			return err
		}
		fallthrough
	case "extract":
		archive := s.silo.GetArchivePath(pkg, version)
		sourceDir := s.silo.GetSourceDirPath(pkg, version)
		if _, err := s.unloadSvc.Unpack(archive, sourceDir); err != nil {
			return err
		}
		fallthrough
	case "build", "rebuild":
		installDir := s.silo.BuildToolPath(pkg, version)
		config := domain.ForgeConfig{
			Name:    pkg,
			Version: version,
			Prefix:  installDir,
			Jobs:    s.jobs,
		}
		_, err := s.forgeSvc.Build(config)
		return err
	}
	return fmt.Errorf("unknown action %q for build tool %s@%s", check.Action, pkg, version)
}

func (s *BundlerService) buildPackage(name, version, phpVersion string, ldPath, cppFlags, ldFlags []string) error {
	check, err := s.advisorSvc.Check(name, version)
	if err != nil {
		return err
	}

	if check.SystemAvailable {
		fmt.Printf("Using system %s@%s at %s\n", name, version, check.SystemPath)
		return nil
	}

	switch check.Action {
	case "skip":
		return nil
	case "download":
		url, err := s.patternRegistry.BuildURLByType(name, version, check.SourceType)
		if err != nil {
			return err
		}
		dest := s.silo.GetArchivePath(name, version)
		if _, err := s.downloadSvc.Download(url, dest); err != nil {
			fmt.Printf("Binary download failed for %s@%s, falling back to source build\n", name, version)
			return s.buildFromSourceOrSystem(name, version, phpVersion, ldPath, cppFlags, ldFlags)
		}
		fallthrough
	case "extract":
		archive := s.silo.GetArchivePath(name, version)
		dest := s.silo.GetSourceDirPath(name, version)
		if _, err := s.unloadSvc.Unpack(archive, dest); err != nil {
			return err
		}
		fallthrough
	case "build", "rebuild":
		return s.compilePackage(name, version, phpVersion, ldPath, cppFlags, ldFlags)
	}
	return fmt.Errorf("unknown action %q for %s@%s", check.Action, name, version)
}

func (s *BundlerService) buildFromSource(name, version, phpVersion string, ldPath, cppFlags, ldFlags []string) error {
	urls := s.getSourceURLs(name, version)

	var lastErr error
	for _, url := range urls {
		dest := s.silo.GetArchivePath(name, version)
		if _, err := s.downloadSvc.Download(url, dest); err != nil {
			lastErr = err
			fmt.Printf("Download failed for %s@%s from %s, trying next mirror...\n", name, version, url)
			continue
		}

		sourceDir := s.silo.GetSourceDirPath(name, version)
		if _, err := s.unloadSvc.Unpack(dest, sourceDir); err != nil {
			lastErr = err
			fmt.Printf("Extraction failed for %s@%s, trying next mirror...\n", name, version)
			continue
		}

		return s.compilePackage(name, version, phpVersion, ldPath, cppFlags, ldFlags)
	}

	if lastErr != nil {
		return fmt.Errorf("all mirrors failed for %s@%s: %w", name, version, lastErr)
	}
	return nil
}

func (s *BundlerService) buildFromSourceOrSystem(name, version, phpVersion string, ldPath, cppFlags, ldFlags []string) error {
	err := s.buildFromSource(name, version, phpVersion, ldPath, cppFlags, ldFlags)
	if err == nil {
		return nil
	}

	check, checkErr := s.advisorSvc.Check(name, version)
	if checkErr != nil {
		return fmt.Errorf("download failed: %w, system check also failed: %v", err, checkErr)
	}

	if check.SystemAvailable {
		fmt.Printf("Using system %s@%s at %s (build from source failed: %v)\n", name, version, check.SystemPath, err)
		return nil
	}

	return err
}

func (s *BundlerService) getSourceURLs(name, version string) []string {
	v := pattern.ParseVersion(version)

	switch name {
	case "libxml2":
		majorMinor := fmt.Sprintf("%d.%d", v.Major, v.Minor)
		return []string{
			fmt.Sprintf("https://download.gnome.org/sources/libxml2/%s/libxml2-%s.tar.xz", majorMinor, version),
			fmt.Sprintf("https://ftp.linux.org.au/pub/gnome.org/sources/libxml2/%s/libxml2-%s.tar.xz", majorMinor, version),
			fmt.Sprintf("https://mirror.freedif.org/GNOME/sources/libxml2/%s/libxml2-%s.tar.xz", majorMinor, version),
		}
	default:
		url, err := s.patternRegistry.BuildURLByType(name, version, domain.SourceTypeSource)
		if err != nil {
			return []string{}
		}
		return []string{url}
	}
}

func (s *BundlerService) compilePackage(name, version, phpVersion string, ldPath, cppFlags, ldFlags []string) error {
	installDir := s.silo.DependencyPath(phpVersion, name, version)

	config := domain.ForgeConfig{
		Name:            name,
		Version:         version,
		Prefix:          installDir,
		Jobs:            s.jobs,
		CPPFLAGS:        cppFlags,
		LDFLAGS:         ldFlags,
		LD_LIBRARY_PATH: ldPath,
	}

	_, err := s.forgeSvc.Build(config)
	return err
}

func (s *BundlerService) buildPHP(name, version string, ldPath, cppFlags, ldFlags []string) error {
	installDir := s.silo.PHPOutputPath(version)
	configureFlags := s.buildPHPConfigureFlags(version, nil)

	config := domain.ForgeConfig{
		Name:            name,
		Version:         version,
		Prefix:          installDir,
		Jobs:            s.jobs,
		CPPFLAGS:        cppFlags,
		LDFLAGS:         ldFlags,
		LD_LIBRARY_PATH: ldPath,
		ConfigureFlags:  configureFlags,
	}

	_, err := s.forgeSvc.Build(config)
	return err
}

type ExtensionConfig struct {
	Name  string
	Path  string
	Flags []string
}

func (s *BundlerService) buildPHPConfigureFlags(phpVersion string, extensions []ExtensionConfig) []string {
	v := pattern.ParseVersion(phpVersion)

	flags := []string{
		"--disable-all",
		"--enable-cli",
		"--with-openssl",
		"--with-curl",
		"--with-zlib",
		"--with-libxml2",
		"--with-onig",
	}

	if v.Major >= 8 {
		flags = append(flags, "--enable-opcache")
	}

	for _, ext := range extensions {
		if ext.Path != "" {
			flags = append(flags, fmt.Sprintf("--with-%s=%s", ext.Name, ext.Path))
		} else {
			flags = append(flags, fmt.Sprintf("--enable-%s", ext.Name))
		}
		flags = append(flags, ext.Flags...)
	}

	return flags
}

func extractVersion(fullVersion string) string {
	if idx := strings.Index(fullVersion, "|"); idx != -1 {
		return fullVersion[:idx]
	}
	return fullVersion
}
