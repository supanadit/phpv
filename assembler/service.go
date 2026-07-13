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
	"github.com/supanadit/phpv/patcher"
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

// ProgressFunc receives human-readable status updates during assembly.
// Pass nil to disable progress reporting.
type ProgressFunc func(stage, message string)

// Service orchestrates the full pipeline:
// resolve deps → download → extract → build deps → build PHP.
type Service struct {
	assemblerRep AssemblerRepository
	registryRep  registry.RegistryRepository
	siloRep      silo.SiloRepository
	forgeRep     forge.ForgeRepository
	patcherRep   patcher.PatcherRepository
}

func NewService(ar AssemblerRepository, rr registry.RegistryRepository, sr silo.SiloRepository, fr forge.ForgeRepository, pr patcher.PatcherRepository) *Service {
	return &Service{
		assemblerRep: ar,
		registryRep:  rr,
		siloRep:      sr,
		forgeRep:     fr,
		patcherRep:   pr,
	}
}

// Assemble runs the full pipeline for (name, version).
// progress (if non-nil) is called with human-readable status updates at each step.
func (s *Service) Assemble(name string, version string, progress ProgressFunc) (*AssemblerResult, error) {
	emit := func(stage, msg string) {
		if progress != nil {
			progress(stage, msg)
		}
	}

	// 1. Resolve the PHP version.
	emit("resolve", fmt.Sprintf("Resolving php version %q...", version))
	exactVersion, err := s.resolveVersion(name, version)
	if err != nil {
		return nil, fmt.Errorf("resolve version: %w", err)
	}
	emit("resolve", fmt.Sprintf("Resolved php %s", exactVersion))

	// Check state file for already-installed.
	statePath := filepath.Join(resolvePHPVRoot("versions"), exactVersion, ".state")
	if data, err := os.ReadFile(statePath); err == nil {
		if strings.TrimSpace(string(data)) == "installed" {
			emit("done", fmt.Sprintf("PHP %s is already installed", exactVersion))
			return &AssemblerResult{PHPVersion: exactVersion}, nil
		}
	}

	// Mark in-progress.
	os.MkdirAll(filepath.Dir(statePath), 0o755)
	os.WriteFile(statePath, []byte("in_progress"), 0o644)

	// Defer marking failed if we return early.
	var completed bool
	defer func() {
		if !completed {
			os.WriteFile(statePath, []byte("failed"), 0o644)
		}
	}()

	// 2. Resolve dependency graph.
	emit("deps", "Resolving dependency graph...")
	deps, err := s.assemblerRep.GetOrderedDependencies(name, exactVersion)
	if err != nil {
		return nil, fmt.Errorf("resolve deps for %s@%s: %w", name, exactVersion, err)
	}
	emit("deps", fmt.Sprintf("Found %d dependencies", len(deps)))

	// 3. Download + extract everything in parallel.
	emit("download", fmt.Sprintf("Downloading and extracting %d packages...", len(deps)+1))
	downloadResults, err := s.downloadAll(name, exactVersion, deps)
	if err != nil {
		return nil, err
	}
	var downloaded, skipped, extracted int
	for _, r := range downloadResults {
		if r.Err == nil {
			if r.Downloaded {
				downloaded++
			} else {
				skipped++
			}
			if r.Extracted {
				extracted++
			}
		}
	}
	emit("download", fmt.Sprintf("Downloaded: %d, Skipped: %d, Extracted: %d", downloaded, skipped, extracted))

	// 4. Build library deps sequentially, accumulating flags.
	var depCppFlags, depLdFlags, depPkgConfigPaths []string
	for _, dep := range deps {
		if isBuildTool(dep.Name) {
			emit("skip", fmt.Sprintf("Skipping build tool %s", dep.Name))
			continue
		}
		depVersion := extractVersion(dep.Version)
		prefix := dependencyPrefix(exactVersion, dep.Name, depVersion)
		sourceDir := filepath.Join(resolvePHPVRoot("sources"), dep.Name, depVersion)

		// Check if this dep is already built and installed.
		if isDepInstalled(prefix) {
			emit("skip", fmt.Sprintf("Already built %s@%s", dep.Name, depVersion))
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
			continue
		}

		emit("build", fmt.Sprintf("Building %s@%s...", dep.Name, depVersion))
		if err := s.applyPatches(dep.Name, depVersion, sourceDir, emit); err != nil {
			emit("error", fmt.Sprintf("Patch failed for %s@%s: %v", dep.Name, depVersion, err))
			return nil, err
		}
		var buildEnv []string
		if flags := s.extraCFlags(dep.Name, depVersion); len(flags) > 0 {
			buildEnv = []string{"CFLAGS=" + strings.Join(flags, " ")}
		}
		extraFlags := s.extraConfigureFlags(dep.Name, depVersion, exactVersion, prefix, sourceDir)
		buildDir, _, err := s.forgeRep.Build(dep.Name, depVersion, sourceDir, buildEnv, extraFlags, prefix)
		if err != nil {
			emit("error", fmt.Sprintf("Build failed for %s@%s", dep.Name, depVersion))
			return nil, fmt.Errorf("forge build %s@%s: %w", dep.Name, depVersion, err)
		}
		emit("install", fmt.Sprintf("Installing %s@%s → %s", dep.Name, depVersion, prefix))
		if err := s.forgeRep.Install(dep.Name, depVersion, buildDir, prefix); err != nil {
			emit("error", fmt.Sprintf("Install failed for %s@%s", dep.Name, depVersion))
			return nil, fmt.Errorf("forge install %s@%s: %w", dep.Name, depVersion, err)
		}
		emit("done", fmt.Sprintf("✓ %s@%s installed", dep.Name, depVersion))

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

	emit("configure", fmt.Sprintf("Configuring %s...", name))
	if err := s.applyPatches(name, exactVersion, phpSrcPath, emit); err != nil {
		emit("error", fmt.Sprintf("Patch failed for %s: %v", name, err))
		return nil, err
	}
	configure := exec.Command("./configure", configureFlags...)
	configure.Dir = phpSrcPath
	configure.Env = env
	if out, err := configure.CombinedOutput(); err != nil {
		emit("error", fmt.Sprintf("Configure failed for %s", name))
		return nil, fmt.Errorf("configure %s@%s: %w\n%s", name, exactVersion, err, out)
	}

	emit("make", fmt.Sprintf("Compiling %s (this may take a while)...", name))
	makeCmd := exec.Command("make", "-j4")
	makeCmd.Dir = phpSrcPath
	makeCmd.Env = env
	if out, err := makeCmd.CombinedOutput(); err != nil {
		emit("error", fmt.Sprintf("Make failed for %s", name))
		return nil, fmt.Errorf("make %s@%s: %w\n%s", name, exactVersion, err, out)
	}

	emit("install", fmt.Sprintf("Installing %s → %s", name, phpPrefix))
	install := exec.Command("make", "install")
	install.Dir = phpSrcPath
	install.Env = env
	if out, err := install.CombinedOutput(); err != nil {
		emit("error", fmt.Sprintf("Install failed for %s", name))
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

	emit("done", fmt.Sprintf("✓ %s %s installed at %s", name, exactVersion, phpPrefix))

	// Mark state as installed.
	completed = true
	os.WriteFile(statePath, []byte("installed"), 0o644)

	return &AssemblerResult{
		DownloadResults: downloadResults,
		PHPVersion:      exactVersion,
		Prefix:          phpPrefix,
		Env: map[string]string{
			"LD_LIBRARY_PATH": strings.Join(depLibraryPaths, ":"),
		},
	}, nil
}

// applyPatches applies all patches for the given package to the extracted source tree.
// Patches are applied in order; if any patch fails, the error is returned immediately.
func (s *Service) applyPatches(name, version, sourceDir string, emit ProgressFunc) error {
	if s.patcherRep == nil {
		return nil
	}
	patches := s.patcherRep.PatchesFor(name, version)
	if len(patches) == 0 {
		return nil
	}
	emit("patch", fmt.Sprintf("Applying %d patch(es) to %s@%s", len(patches), name, version))
	for _, p := range patches {
		emit("patch", fmt.Sprintf("  → %s", p.Name))
		if p.Apply != nil {
			if err := p.Apply(sourceDir); err != nil {
				return fmt.Errorf("patch %s: %w", p.Name, err)
			}
		}
	}
	return nil
}

// extraCFlags returns package-specific CFLAGS needed to build old code on
// modern toolchains. Empty if no special flags are required.
func (s *Service) extraCFlags(name, version string) []string {
	if s.patcherRep == nil {
		return nil
	}
	patches := s.patcherRep.PatchesFor(name, version)
	for _, p := range patches {
		if p.ExtraCFlags != nil {
			return p.ExtraCFlags
		}
	}
	return nil
}

// extraConfigureFlags resolves patch-level configure flag templates against
// the package's install prefix and source directory. Placeholders:
//   {{prefix}} → install prefix
//   {{source}} → source dir
//   {{dep:NAME}} → install prefix of dependency NAME (under phpVersion)
func (s *Service) extraConfigureFlags(name, depVersion, phpVersion, prefix, sourceDir string) []string {
	if s.patcherRep == nil {
		return nil
	}
	patches := s.patcherRep.PatchesFor(name, depVersion)
	for _, p := range patches {
		if len(p.ConfigureFlags) == 0 {
			continue
		}
		resolved := make([]string, 0, len(p.ConfigureFlags))
		for _, flag := range p.ConfigureFlags {
			flag = strings.ReplaceAll(flag, "{{prefix}}", prefix)
			flag = strings.ReplaceAll(flag, "{{source}}", sourceDir)
			// Resolve {{dep:NAME}} → the dep's install prefix.
			if strings.Contains(flag, "{{dep:") {
				start := strings.Index(flag, "{{dep:")
				end := strings.Index(flag[start:], "}}")
				if end > 0 {
					depName := flag[start+6 : start+end]
					depBase := filepath.Join(resolvePHPVRoot("versions"), phpVersion, "dependency", depName)
					depPrefix := filepath.Join(depBase, "1.0.0")
					entries, _ := os.ReadDir(depBase)
					if len(entries) > 0 {
						depPrefix = filepath.Join(depBase, entries[0].Name())
					}
					flag = strings.ReplaceAll(flag, "{{dep:"+depName+"}}", depPrefix)
				}
			}
			resolved = append(resolved, flag)
		}
		return resolved
	}
	return nil
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

// isBuildTool returns true if the package is a build tool that should be
// skipped during source build (we use system-installed versions).
func isBuildTool(name string) bool {
	switch name {
	case "m4", "autoconf", "automake", "libtool", "perl", "bison", "re2c", "flex", "zig":
		return true
	}
	return false
}

// isDepInstalled checks whether a dependency is already built and installed
// at the given prefix by looking for the include/ directory.
func isDepInstalled(prefix string) bool {
	info, err := os.Stat(filepath.Join(prefix, "include"))
	if err != nil {
		return false
	}
	return info.IsDir()
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
