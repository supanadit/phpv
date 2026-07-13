package assembler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/forge"
	"github.com/supanadit/phpv/registry"
	"github.com/supanadit/phpv/silo"
)

// AssemblerRepository resolves the transitive dependency graph.
type AssemblerRepository interface {
	GetOrderedDependencies(name string, version string) ([]domain.Dependency, error)
}

// AssemblerResult holds the outcome of assembling a package.
type AssemblerResult struct {
	DownloadResults []DownloadResult
	PHPVersion      string
	Prefix          string
	Env             map[string]string
}

// DownloadResult holds the outcome of downloading + extracting a single package.
type DownloadResult struct {
	Name       string
	Version    string
	Downloaded bool
	Extracted  bool
	Err        error
}

// Service orchestrates the full pipeline:
// resolve deps → download → extract → build deps → build PHP.
type Service struct {
	assemblerRep AssemblerRepository
	registryRep  registry.RegistryRepository
	siloRep      silo.SiloRepository
	forgeRep     forge.ForgeRepository
}

func NewService(ar AssemblerRepository, rr registry.RegistryRepository, sr silo.SiloRepository, fr forge.ForgeRepository) *Service {
	return &Service{
		assemblerRep: ar,
		registryRep:  rr,
		siloRep:      sr,
		forgeRep:     fr,
	}
}

// Assemble runs the full pipeline for (name, version).
func (s *Service) Assemble(name string, version string) (*AssemblerResult, error) {
	// 1. Resolve exact version.
	exactVersion, err := s.resolveVersion(name, version)
	if err != nil {
		return nil, fmt.Errorf("resolve version %q: %w", version, err)
	}

	// 2. Resolve dependency graph.
	deps, err := s.assemblerRep.GetOrderedDependencies(name, exactVersion)
	if err != nil {
		return nil, fmt.Errorf("resolve deps for %s@%s: %w", name, exactVersion, err)
	}

	// 3. Download + extract everything in parallel.
	downloadResults, err := s.downloadAll(name, exactVersion, deps)
	if err != nil {
		return nil, err
	}

	// 4. Build library deps sequentially, accumulating flags.
	var depCppFlags, depLdFlags, depPkgConfigPaths []string
	for _, dep := range deps {
		if isBuildTool(dep.Name) {
			continue
		}
		depVersion := extractVersion(dep.Version)
		prefix := dependencyPrefix(exactVersion, dep.Name, depVersion)
		sourceDir := filepath.Join(resolvePHPVRoot("sources"), dep.Name, depVersion)

		buildDir, _, err := s.forgeRep.Build(dep.Name, depVersion, sourceDir, nil)
		if err != nil {
			return nil, fmt.Errorf("forge build %s@%s: %w", dep.Name, depVersion, err)
		}
		if err := s.forgeRep.Install(dep.Name, depVersion, buildDir, prefix); err != nil {
			return nil, fmt.Errorf("forge install %s@%s: %w", dep.Name, depVersion, err)
		}

		depPath := prefix
		depCppFlags = appendUnique(depCppFlags, "-I"+filepath.Join(depPath, "include"))
		depLdFlags = appendUnique(depLdFlags, "-L"+filepath.Join(depPath, "lib"))
		depLdFlags = appendUnique(depLdFlags, "-Wl,-rpath,"+filepath.Join(depPath, "lib"))

		pcPath := filepath.Join(depPath, "lib", "pkgconfig")
		if _, err := os.Stat(pcPath); err == nil {
			depPkgConfigPaths = appendUnique(depPkgConfigPaths, pcPath)
		}
		pc64Path := filepath.Join(depPath, "lib64", "pkgconfig")
		if _, err := os.Stat(pc64Path); err == nil {
			depPkgConfigPaths = appendUnique(depPkgConfigPaths, pc64Path)
		}
	}

	// 5. Build PHP.
	phpPrefix := phpOutputPath(exactVersion)
	phpSourceDir := filepath.Join(resolvePHPVRoot("sources"), name, exactVersion)
	phpSrcPath := findSourceDir(phpSourceDir, name, exactVersion)
	if phpSrcPath == "" {
		return nil, fmt.Errorf("could not find source directory in %s", phpSourceDir)
	}

	configureFlags := []string{
		"--prefix=" + phpPrefix,
		"--with-pdo-mysql=mysqlnd",
		"--with-mysqli=mysqlnd",
	}

	env := os.Environ()
	if len(depCppFlags) > 0 {
		env = setEnvVar(env, "CPPFLAGS", strings.Join(depCppFlags, " "))
	}
	if len(depLdFlags) > 0 {
		env = setEnvVar(env, "LDFLAGS", strings.Join(depLdFlags, " "))
	}
	if len(depPkgConfigPaths) > 0 {
		env = setEnvVar(env, "PKG_CONFIG_PATH", strings.Join(depPkgConfigPaths, ":"))
	}

	configure := exec.Command("./configure", configureFlags...)
	configure.Dir = phpSrcPath
	configure.Env = env
	if out, err := configure.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("configure %s@%s: %w\n%s", name, exactVersion, err, out)
	}

	makeCmd := exec.Command("make", "-j4")
	makeCmd.Dir = phpSrcPath
	makeCmd.Env = env
	if out, err := makeCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("make %s@%s: %w\n%s", name, exactVersion, err, out)
	}

	install := exec.Command("make", "install")
	install.Dir = phpSrcPath
	install.Env = env
	if out, err := install.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("make install %s@%s: %w\n%s", name, exactVersion, err, out)
	}

	var depLibraryPaths []string
	for _, dep := range deps {
		if isBuildTool(dep.Name) {
			continue
		}
		depVersion := extractVersion(dep.Version)
		depLibraryPaths = appendUnique(depLibraryPaths, filepath.Join(dependencyPrefix(exactVersion, dep.Name, depVersion), "lib"))
	}
	depLibraryPaths = appendUnique(depLibraryPaths, filepath.Join(phpPrefix, "lib"))

	return &AssemblerResult{
		DownloadResults: downloadResults,
		PHPVersion:      exactVersion,
		Prefix:          phpPrefix,
		Env: map[string]string{
			"LD_LIBRARY_PATH": strings.Join(depLibraryPaths, ":"),
		},
	}, nil
}

// downloadAll downloads + extracts the root package and all deps in parallel.
func (s *Service) downloadAll(name, version string, deps []domain.Dependency) ([]DownloadResult, error) {
	type item struct {
		name    string
		version string
	}
	var items []item
	for _, dep := range deps {
		items = append(items, item{dep.Name, extractVersion(dep.Version)})
	}
	items = append(items, item{name, version})

	// dedup
	seen := make(map[string]bool)
	var unique []item
	for _, it := range items {
		key := it.name + "@" + it.version
		if seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, it)
	}
	items = unique

	results := make([]DownloadResult, len(items))
	var wg sync.WaitGroup

	for i, it := range items {
		wg.Add(1)
		go func(idx int, n, v string) {
			defer wg.Done()
			results[idx] = DownloadResult{Name: n, Version: v}

			r, err := s.registryRep.Get(n, v, false, runtime.GOOS)
			if err != nil {
				results[idx].Err = fmt.Errorf("registry resolve %s@%s: %w", n, v, err)
				return
			}
			downloaded, err := s.siloRep.Download(r.URL, r.ChecksumType, r.ChecksumValue)
			if err != nil {
				results[idx].Err = fmt.Errorf("download %s@%s: %w", n, v, err)
				return
			}
			results[idx].Downloaded = downloaded

			archivePath := filepath.Join(cacheDir(), filepath.Base(r.URL))
			sourceDir := filepath.Join(sourcesDir(), n, v)
			extracted, err := s.siloRep.Extract(archivePath, sourceDir)
			if err != nil {
				results[idx].Err = fmt.Errorf("extract %s@%s: %w", n, v, err)
				return
			}
			results[idx].Extracted = extracted
		}(i, it.name, it.version)
	}

	wg.Wait()
	for _, r := range results {
		if r.Err != nil {
			return results, fmt.Errorf("one or more downloads failed")
		}
	}
	return results, nil
}

func (s *Service) resolveVersion(name, constraint string) (string, error) {
	entries, err := s.registryRep.List(name, false, "")
	if err != nil {
		return "", err
	}
	var versions []string
	for _, e := range entries {
		versions = append(versions, e.Version)
	}
	return resolveVersionConstraint(versions, constraint)
}

// Helpers.

func isBuildTool(name string) bool {
	switch name {
	case "m4", "autoconf", "automake", "libtool", "perl", "bison", "flex", "re2c", "cmake":
		return true
	default:
		return false
	}
}

func phpOutputPath(phpVersion string) string {
	return filepath.Join(resolvePHPVRoot("versions"), phpVersion, "output")
}

func dependencyPrefix(phpVersion, name, version string) string {
	return filepath.Join(resolvePHPVRoot("versions"), phpVersion, "dependency", name, version)
}

func resolvePHPVRoot(parts ...string) string {
	root := os.Getenv("PHPV_ROOT")
	if root == "" {
		home, _ := os.UserHomeDir()
		root = filepath.Join(home, ".phpv")
	}
	return filepath.Join(append([]string{root}, parts...)...)
}

func cacheDir() string   { return resolvePHPVRoot("caches") }
func sourcesDir() string { return resolvePHPVRoot("sources") }

func appendUnique(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}

func extractVersion(v string) string {
	if v == "" {
		return ""
	}
	if before, _, found := strings.Cut(v, "|"); found {
		return before
	}
	return v
}

func setEnvVar(env []string, key, value string) []string {
	prefix := key + "="
	for i, v := range env {
		if strings.HasPrefix(v, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

func findSourceDir(extractDir, name, version string) string {
	if hasBuildFile(extractDir) {
		return extractDir
	}
	entries, err := os.ReadDir(extractDir)
	if err != nil {
		return ""
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			dirs = append(dirs, e.Name())
		}
	}
	if len(dirs) == 1 {
		candidate := filepath.Join(extractDir, dirs[0])
		if hasBuildFile(candidate) {
			return candidate
		}
	}
	for _, d := range dirs {
		candidate := filepath.Join(extractDir, d)
		if hasBuildFile(candidate) {
			return candidate
		}
	}
	return ""
}

func hasBuildFile(dir string) bool {
	return fileExists(filepath.Join(dir, "configure")) ||
		fileExists(filepath.Join(dir, "Configure")) ||
		fileExists(filepath.Join(dir, "CMakeLists.txt")) ||
		fileExists(filepath.Join(dir, "Makefile")) ||
		fileExists(filepath.Join(dir, "makefile"))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func resolveVersionConstraint(versions []string, constraint string) (string, error) {
	for _, v := range versions {
		if v == constraint {
			return v, nil
		}
	}
	if strings.Count(constraint, ".") == 1 {
		prefix := constraint + "."
		var candidates []string
		for _, v := range versions {
			if strings.HasPrefix(v, prefix) {
				candidates = append(candidates, v)
			}
		}
		if len(candidates) > 0 {
			best := candidates[0]
			for _, v := range candidates[1:] {
				if compareVersions(v, best) > 0 {
					best = v
				}
			}
			return best, nil
		}
	}
	return "", fmt.Errorf("no version matching %q found", constraint)
}

func compareVersions(a, b string) int {
	ap := strings.Split(a, ".")
	bp := strings.Split(b, ".")
	for i := 0; i < 3; i++ {
		var an, bn int
		if i < len(ap) {
			fmt.Sscanf(ap[i], "%d", &an)
		}
		if i < len(bp) {
			fmt.Sscanf(bp[i], "%d", &bn)
		}
		if an != bn {
			if an > bn {
				return 1
			}
			return -1
		}
	}
	return 0
}

// DownloadFailed returns true if any result has an error.
func DownloadFailed(results []DownloadResult) bool {
	for _, r := range results {
		if r.Err != nil {
			return true
		}
	}
	return false
}
