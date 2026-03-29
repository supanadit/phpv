package disk

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/afero"
	"github.com/supanadit/phpv/advisor"
	"github.com/supanadit/phpv/assembler"
	"github.com/supanadit/phpv/bundler"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/download"
	"github.com/supanadit/phpv/forge"
	"github.com/supanadit/phpv/pattern"
	"github.com/supanadit/phpv/source"
	"github.com/supanadit/phpv/unload"
)

type bundlerRepository struct {
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

func NewBundlerRepository(cfg bundler.BundlerServiceConfig) bundler.BundlerRepository {
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

	return &bundlerRepository{
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

func (s *bundlerRepository) Install(version string) (domain.Forge, error) {
	exactVersion, err := s.resolvePHPVersion(version)
	if err != nil {
		return domain.Forge{}, fmt.Errorf("failed to resolve version %q: %w", version, err)
	}
	return s.Orchestrate("php", exactVersion)
}

func (s *bundlerRepository) resolvePHPVersion(constraint string) (string, error) {
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

func (s *bundlerRepository) Orchestrate(name, exactVersion string) (domain.Forge, error) {
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

func (s *bundlerRepository) ensureBuildTools() error {
	buildTools := []struct {
		pkg string
		ver string
	}{
		{"m4", "1.4.19"},
		{"autoconf", "2.69"},
		{"autoconf", "2.71"},
		{"autoconf", "2.72"},
		{"automake", "1.16.5"},
		{"automake", "1.17"},
		{"libtool", "2.4.7"},
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

		if err := s.installBuildTool(tool.pkg, tool.ver, true); err != nil {
			return fmt.Errorf("failed to install build tool %s@%s: %w", tool.pkg, tool.ver, err)
		}
	}

	return nil
}

func (s *bundlerRepository) installBuildTool(pkg, version string, forceSource bool) error {
	if !forceSource {
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

	sourceType := domain.SourceTypeSource
	url, err := s.patternRegistry.BuildURLByType(pkg, version, sourceType)
	if err != nil {
		return err
	}
	archive := archivePathFromURL(s.silo.Root, pkg, version, url)
	if _, err := s.downloadSvc.Download(url, archive); err != nil {
		return err
	}

	sourceDir := s.silo.GetSourceDirPath(pkg, version)
	if _, err := s.unloadSvc.Unpack(archive, sourceDir); err != nil {
		return err
	}

	installDir := s.silo.BuildToolPath(pkg, version)
	config := domain.ForgeConfig{
		Name:    pkg,
		Version: version,
		Prefix:  installDir,
		Jobs:    s.jobs,
	}
	return s.forgePkg(config, sourceDir, url)
}

func (s *bundlerRepository) forgePkg(config domain.ForgeConfig, sourceDir, url string) error {
	s.fs.MkdirAll(sourceDir, 0o755)

	env := s.buildEnvForForge(config, s.silo.Root)

	strategy := s.detectForgeStrategy(config.Name)
	switch strategy {
	case domain.StrategyCMake:
		return s.forgeBuildCMake(sourceDir, config.Prefix, config, env)
	case domain.StrategyMakeOnly:
		return s.forgeBuildMakeOnly(sourceDir, config.Prefix, config, env)
	case domain.StrategyConfigureMake:
		return s.forgeBuildConfigureMake(sourceDir, config.Prefix, config, env)
	case domain.StrategyAutogen:
		return s.forgeBuildAutogen(sourceDir, config.Prefix, config, env)
	default:
		return fmt.Errorf("unsupported build strategy: %s", strategy)
	}
}

func (s *bundlerRepository) detectForgeStrategy(name string) domain.BuildStrategy {
	switch name {
	case "zlib":
		return domain.StrategyMakeOnly
	case "cmake":
		return domain.StrategyCMake
	case "autoconf", "automake", "flex", "bison", "perl":
		return domain.StrategyMakeOnly
	case "openssl":
		return domain.StrategyConfigureMake
	case "php":
		return domain.StrategyConfigureMake
	default:
		return domain.StrategyConfigureMake
	}
}

func (s *bundlerRepository) buildEnvForForge(config domain.ForgeConfig, root string) []string {
	env := os.Environ()

	buildToolsPath := filepath.Join(root, "build-tools")
	buildToolsBinPath := s.collectBuildToolsBinPaths(buildToolsPath)

	for i, v := range env {
		if strings.HasPrefix(v, "PATH=") {
			env[i] = "PATH=" + buildToolsBinPath + ":" + strings.TrimPrefix(v, "PATH=")
			break
		}
	}

	for _, v := range config.CPPFLAGS {
		env = append(env, "CPPFLAGS="+v)
	}
	for _, v := range config.LDFLAGS {
		env = append(env, "LDFLAGS="+v)
	}
	if len(config.LD_LIBRARY_PATH) > 0 {
		env = append(env, "LD_LIBRARY_PATH="+strings.Join(config.LD_LIBRARY_PATH, ":"))
	}
	for k, v := range config.Env {
		env = append(env, k+"="+v)
	}

	return env
}

func (s *bundlerRepository) collectBuildToolsBinPaths(buildToolsPath string) string {
	var binPaths []string

	entries, _ := afero.ReadDir(s.fs, buildToolsPath)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pkgPath := filepath.Join(buildToolsPath, entry.Name())
		versionEntries, _ := afero.ReadDir(s.fs, pkgPath)
		for _, vEntry := range versionEntries {
			if !vEntry.IsDir() {
				continue
			}
			binPath := filepath.Join(pkgPath, vEntry.Name(), "bin")
			if exists, _ := afero.DirExists(s.fs, binPath); exists {
				binPaths = append(binPaths, binPath)
			}
		}
	}

	return strings.Join(binPaths, ":")
}

func (s *bundlerRepository) forgeBuildConfigureMake(sourcePath, prefix string, config domain.ForgeConfig, env []string) error {
	configurePath := filepath.Join(sourcePath, "configure")
	if _, err := os.Stat(configurePath); os.IsNotExist(err) {
		return fmt.Errorf("configure script not found at %s", configurePath)
	}

	if err := os.Chmod(configurePath, 0o755); err != nil {
		return fmt.Errorf("failed to chmod configure: %w", err)
	}

	args := []string{fmt.Sprintf("--prefix=%s", prefix)}
	args = append(args, config.ConfigureFlags...)

	configure := exec.Command("./configure", args...)
	configure.Dir = sourcePath
	configure.Env = env
	configure.Stdout = os.Stdout
	configure.Stderr = os.Stderr

	fmt.Println("Running configure for", config.Name)
	if err := configure.Run(); err != nil {
		return fmt.Errorf("configure failed: %w", err)
	}

	if err := s.forgeMake(sourcePath, config.Name, config.Jobs, env); err != nil {
		return err
	}

	return s.forgeMakeInstall(sourcePath, config.Name, config.Jobs, env)
}

func (s *bundlerRepository) forgeBuildMakeOnly(sourcePath, prefix string, config domain.ForgeConfig, env []string) error {
	if err := s.forgeMake(sourcePath, config.Name, config.Jobs, env); err != nil {
		return fmt.Errorf("make failed: %w", err)
	}

	return s.forgeMakeInstall(sourcePath, config.Name, config.Jobs, env)
}

func (s *bundlerRepository) forgeBuildCMake(sourcePath, prefix string, config domain.ForgeConfig, env []string) error {
	buildDir := filepath.Join(sourcePath, "build")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return fmt.Errorf("failed to create build directory: %w", err)
	}

	jobs := config.Jobs
	if jobs == 0 {
		jobs = runtime.NumCPU()
	}

	cmakeArgs := []string{
		"-DCMAKE_INSTALL_PREFIX=" + prefix,
		sourcePath,
	}

	cmakeCmd := exec.Command("cmake", cmakeArgs...)
	cmakeCmd.Dir = buildDir
	cmakeCmd.Env = env
	cmakeCmd.Stdout = os.Stdout
	cmakeCmd.Stderr = os.Stderr

	fmt.Println("Running cmake for", config.Name)
	if err := cmakeCmd.Run(); err != nil {
		return fmt.Errorf("cmake failed: %w", err)
	}

	mk := exec.Command("make", fmt.Sprintf("-j%d", jobs))
	mk.Dir = buildDir
	mk.Env = env
	mk.Stdout = os.Stdout
	mk.Stderr = os.Stderr

	fmt.Println("Running make for", config.Name)
	if err := mk.Run(); err != nil {
		return fmt.Errorf("make failed: %w", err)
	}

	return s.forgeMakeInstall(buildDir, config.Name, config.Jobs, env)
}

func (s *bundlerRepository) forgeBuildAutogen(sourcePath, prefix string, config domain.ForgeConfig, env []string) error {
	autogenPath := filepath.Join(sourcePath, "autogen.sh")
	if _, err := os.Stat(autogenPath); err == nil {
		autogen := exec.Command("./autogen.sh")
		autogen.Dir = sourcePath
		autogen.Env = env
		autogen.Stdout = os.Stdout
		autogen.Stderr = os.Stderr
		fmt.Println("Running autogen.sh for", config.Name)
		if err := autogen.Run(); err != nil {
			return fmt.Errorf("autogen failed: %w", err)
		}
	}

	configurePath := filepath.Join(sourcePath, "configure")
	if _, err := os.Stat(configurePath); err == nil {
		if err := os.Chmod(configurePath, 0o755); err != nil {
			return fmt.Errorf("failed to chmod configure: %w", err)
		}

		args := []string{fmt.Sprintf("--prefix=%s", prefix)}
		args = append(args, config.ConfigureFlags...)

		configure := exec.Command("./configure", args...)
		configure.Dir = sourcePath
		configure.Env = env
		configure.Stdout = os.Stdout
		configure.Stderr = os.Stderr

		fmt.Println("Running configure for", config.Name)
		if err := configure.Run(); err != nil {
			return fmt.Errorf("configure failed: %w", err)
		}
	}

	if err := s.forgeMake(sourcePath, config.Name, config.Jobs, env); err != nil {
		return err
	}

	return s.forgeMakeInstall(sourcePath, config.Name, config.Jobs, env)
}

func (s *bundlerRepository) forgeMake(sourcePath, pkgName string, jobs int, env []string) error {
	if jobs == 0 {
		jobs = runtime.NumCPU()
	}

	mk := exec.Command("make", fmt.Sprintf("-j%d", jobs))
	mk.Dir = sourcePath
	mk.Env = env
	mk.Stdout = os.Stdout
	mk.Stderr = os.Stderr

	fmt.Println("Running make for", pkgName)
	if err := mk.Run(); err != nil {
		return fmt.Errorf("make failed: %w", err)
	}

	return nil
}

func (s *bundlerRepository) forgeMakeInstall(sourcePath, pkgName string, jobs int, env []string) error {
	mkInstall := exec.Command("make", "install")
	mkInstall.Dir = sourcePath
	mkInstall.Env = env
	mkInstall.Stdout = os.Stdout
	mkInstall.Stderr = os.Stderr

	fmt.Println("Running make install for", pkgName)
	if err := mkInstall.Run(); err != nil {
		return fmt.Errorf("make install failed: %w", err)
	}

	return nil
}

func (s *bundlerRepository) buildPackage(name, version, phpVersion string, ldPath, cppFlags, ldFlags []string) error {
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
		archive := archivePathFromURL(s.silo.Root, name, version, url)
		if _, err := s.downloadSvc.Download(url, archive); err != nil {
			fmt.Printf("Binary download failed for %s@%s, falling back to source build\n", name, version)
			return s.buildFromSourceOrSystem(name, version, phpVersion, ldPath, cppFlags, ldFlags)
		}
		fallthrough
	case "extract":
		url, err := s.patternRegistry.BuildURLByType(name, version, check.SourceType)
		if err != nil {
			return err
		}
		archive := archivePathFromURL(s.silo.Root, name, version, url)
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

func (s *bundlerRepository) buildFromSource(name, version, phpVersion string, ldPath, cppFlags, ldFlags []string) error {
	urls := s.getSourceURLs(name, version)

	var lastErr error
	for _, url := range urls {
		archive := archivePathFromURL(s.silo.Root, name, version, url)
		if _, err := s.downloadSvc.Download(url, archive); err != nil {
			lastErr = err
			fmt.Printf("Download failed for %s@%s from %s, trying next mirror...\n", name, version, url)
			continue
		}

		sourceDir := s.silo.GetSourceDirPath(name, version)
		if _, err := s.unloadSvc.Unpack(archive, sourceDir); err != nil {
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

func (s *bundlerRepository) buildFromSourceOrSystem(name, version, phpVersion string, ldPath, cppFlags, ldFlags []string) error {
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

func (s *bundlerRepository) getSourceURLs(name, version string) []string {
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

func (s *bundlerRepository) compilePackage(name, version, phpVersion string, ldPath, cppFlags, ldFlags []string) error {
	installDir := s.silo.DependencyPath(phpVersion, name, version)

	config := domain.ForgeConfig{
		Name:            name,
		Version:         version,
		Prefix:          installDir,
		Jobs:            s.jobs,
		CPPFLAGS:        cppFlags,
		LDFLAGS:         ldFlags,
		LD_LIBRARY_PATH: ldPath,
		ConfigureFlags:  s.getConfigureFlags(name),
	}

	_, err := s.forgeSvc.Build(config)
	return err
}

func (s *bundlerRepository) getConfigureFlags(name string) []string {
	switch name {
	case "m4":
		return []string{"--disable-maintainer-mode"}
	default:
		return nil
	}
}

func (s *bundlerRepository) buildPHP(name, version string, ldPath, cppFlags, ldFlags []string) error {
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

func (s *bundlerRepository) buildPHPConfigureFlags(phpVersion string, extensions []ExtensionConfig) []string {
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

func archivePathFromURL(root, pkg, ver, url string) string {
	filename := filepath.Base(url)
	return filepath.Join(root, "cache", pkg, ver, filename)
}
