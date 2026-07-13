package assembler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/forge"
	"github.com/supanadit/phpv/graph"
	"github.com/supanadit/phpv/patcher"
	"github.com/supanadit/phpv/registry"
	"github.com/supanadit/phpv/silo"
)

// AssemblerResult holds the outcome of assembling a package.
type AssemblerResult struct {
	DownloadResults []DownloadResult
	Version         string
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

// Service orchestrates the full build pipeline for any package:
// resolve → get build plan → download → build deps → build package → mark complete.
// It composes service wrappers that each add value over their underlying repositories.
type Service struct {
	graph   *graph.Service
	silo    *silo.Service
	forge   *forge.Service
	patcher *patcher.Service
	reg     *registry.Service
}

// NewService creates a new assembler service.
func NewService(g *graph.Service, s *silo.Service, f *forge.Service, p *patcher.Service, r *registry.Service) *Service {
	return &Service{
		graph:   g,
		silo:    s,
		forge:   f,
		patcher: p,
		reg:     r,
	}
}

// Graph returns the graph service used by the assembler.
func (s *Service) Graph() *graph.Service {
	return s.graph
}

// Assemble runs the full pipeline for (name, version).
func (s *Service) Assemble(name string, version string, static bool, extensions []string, progress ProgressFunc) (*AssemblerResult, error) {
	emit := func(stage, msg string) {
		if progress != nil {
			progress(stage, msg)
		}
	}

	emit("resolve", fmt.Sprintf("Resolving %s version %q...", name, version))
	exactVersion, err := s.resolveVersion(name, version)
	if err != nil {
		return nil, fmt.Errorf("resolve version: %w", err)
	}
	emit("resolve", fmt.Sprintf("Resolved %s %s", name, exactVersion))

	prefix := s.silo.PackagePrefix(name, exactVersion)

	state, err := s.silo.GetState(name, exactVersion)
	if err != nil {
		return nil, fmt.Errorf("get state: %w", err)
	}
	if state == domain.StateInstalled {
		emit("done", fmt.Sprintf("%s %s is already installed", name, exactVersion))
		return &AssemblerResult{Version: exactVersion}, nil
	}

	if err := s.silo.MarkInProgress(name, exactVersion); err != nil {
		return nil, fmt.Errorf("mark in-progress: %w", err)
	}

	var completed bool
	defer func() {
		if !completed {
			s.silo.MarkFailed(name, exactVersion)
		}
	}()

	emit("deps", "Resolving dependency graph...")
	plan, err := s.graph.GetBuildPlan(name, exactVersion, extensions)
	if err != nil {
		return nil, fmt.Errorf("resolve build plan for %s@%s: %w", name, exactVersion, err)
	}
	emit("deps", fmt.Sprintf("Found %d dependencies", len(plan.Deps)))

	emit("download", fmt.Sprintf("Downloading and extracting %d packages...", len(plan.Deps)+1))
	downloadResults, err := s.downloadAll(name, exactVersion, plan.Deps)
	if err != nil {
		return nil, err
	}
	var downloaded, skipped, extracted int
	for _, dr := range downloadResults {
		if dr.Err == nil {
			if dr.Downloaded {
				downloaded++
			} else {
				skipped++
			}
			if dr.Extracted {
				extracted++
			}
		}
	}
	emit("download", fmt.Sprintf("Downloaded: %d, Skipped: %d, Extracted: %d", downloaded, skipped, extracted))

	var depCppFlags, depLdFlags, depPkgConfigPaths []string
	for _, dep := range plan.Deps {
		if isBuildTool(dep.Name) {
			emit("skip", fmt.Sprintf("Skipping build tool %s", dep.Name))
			continue
		}
		depVersion := extractVersion(dep.Version)
		depPrefix := s.silo.PackagePrefix(dep.Name, depVersion)
		sourceDir := s.silo.SourcePath(dep.Name, depVersion)

		if isDepInstalled(depPrefix) {
			emit("skip", fmt.Sprintf("Already built %s@%s", dep.Name, depVersion))
			depCppFlags, depLdFlags, depPkgConfigPaths = s.collectDepFlags(depPrefix, depCppFlags, depLdFlags, depPkgConfigPaths)
			continue
		}

		emit("build", fmt.Sprintf("Building %s@%s...", dep.Name, depVersion))
		prepared, err := s.patcher.Prepare(dep.Name, depVersion, sourceDir)
		if err != nil {
			emit("error", fmt.Sprintf("Patch failed for %s@%s: %v", dep.Name, depVersion, err))
			return nil, err
		}
		if len(prepared.Applied) > 0 {
			emit("patch", fmt.Sprintf("Applied %d patch(es) to %s@%s", len(prepared.Applied), dep.Name, depVersion))
		}
		var buildEnv []string
		if len(prepared.ExtraCFlags) > 0 {
			buildEnv = []string{"CFLAGS=" + strings.Join(prepared.ExtraCFlags, " ")}
		}
		buildDir, _, err := s.forge.Build(dep.Name, depVersion, sourceDir, buildEnv, prepared.ConfigureFlags, depPrefix)
		if err != nil {
			emit("error", fmt.Sprintf("Build failed for %s@%s", dep.Name, depVersion))
			return nil, fmt.Errorf("forge build %s@%s: %w", dep.Name, depVersion, err)
		}
		emit("install", fmt.Sprintf("Installing %s@%s → %s", dep.Name, depVersion, depPrefix))
		if err := s.forge.Install(dep.Name, depVersion, buildDir, depPrefix); err != nil {
			emit("error", fmt.Sprintf("Install failed for %s@%s", dep.Name, depVersion))
			return nil, fmt.Errorf("forge install %s@%s: %w", dep.Name, depVersion, err)
		}
		emit("done", fmt.Sprintf("✓ %s@%s installed", dep.Name, depVersion))

		depCppFlags, depLdFlags, depPkgConfigPaths = s.collectDepFlags(depPrefix, depCppFlags, depLdFlags, depPkgConfigPaths)
	}

	sourceDir := s.silo.SourcePath(name, exactVersion)
	srcPath := FindSourceDir(sourceDir, name, exactVersion)
	if srcPath == "" {
		return nil, fmt.Errorf("could not find source directory in %s", sourceDir)
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
	if len(plan.CompilerFlags) > 0 {
		env = setEnvVar(env, "CFLAGS", strings.Join(plan.CompilerFlags, " "))
		env = setEnvVar(env, "CXXFLAGS", strings.Join(plan.CompilerFlags, " "))
	}

	configureFlags := plan.ConfigureFlags
	if static {
		env = setEnvVar(env, "LDFLAGS", "-static-libgcc -static")
		env = setEnvVar(env, "CFLAGS", "-Os -fdata-sections -ffunction-sections")
		env = setEnvVar(env, "CXXFLAGS", "-Os -fdata-sections -ffunction-sections")
		configureFlags = append([]string{"--enable-static", "--disable-shared"}, configureFlags...)
	}

	emit("configure", fmt.Sprintf("Configuring %s...", name))
	prepared, err := s.patcher.Prepare(name, exactVersion, srcPath)
	if err != nil {
		emit("error", fmt.Sprintf("Patch failed for %s: %v", name, err))
		return nil, err
	}
	if len(prepared.Applied) > 0 {
		emit("patch", fmt.Sprintf("Applied %d patch(es) to %s", len(prepared.Applied), name))
	}

	emit("make", fmt.Sprintf("Compiling %s (this may take a while)...", name))
	buildDir, _, err := s.forge.Build(name, exactVersion, srcPath, env, plan.ConfigureFlags, prefix)
	if err != nil {
		emit("error", fmt.Sprintf("Build failed for %s", name))
		return nil, fmt.Errorf("build %s@%s: %w", name, exactVersion, err)
	}
	if err := s.forge.Install(name, exactVersion, buildDir, prefix); err != nil {
		emit("error", fmt.Sprintf("Install failed for %s", name))
		return nil, fmt.Errorf("install %s@%s: %w", name, exactVersion, err)
	}

	emit("done", fmt.Sprintf("✓ %s %s installed at %s", name, exactVersion, prefix))

	if len(extensions) > 0 {
		emit("extensions", fmt.Sprintf("Building %d extension(s)...", len(extensions)))
		sourceDir := s.silo.SourcePath(name, exactVersion)
		srcPath := FindSourceDir(sourceDir, name, exactVersion)
		if srcPath == "" {
			return nil, fmt.Errorf("could not find source directory for extensions in %s", sourceDir)
		}
		for _, ext := range extensions {
			emit("extensions", fmt.Sprintf("Building extension %s...", ext))
			if err := s.InstallExtension(exactVersion, ext, srcPath, prefix); err != nil {
				emit("error", fmt.Sprintf("Extension %s failed: %v", ext, err))
				return nil, fmt.Errorf("extension %s: %w", ext, err)
			}
			emit("extensions", fmt.Sprintf("✓ %s built", ext))
		}
	}

	var depLibraryPaths []string
	for _, dep := range plan.Deps {
		if isBuildTool(dep.Name) {
			continue
		}
		depVersion := extractVersion(dep.Version)
		depLibraryPaths = appendUnique(depLibraryPaths, filepath.Join(s.silo.PackagePrefix(dep.Name, depVersion), "lib"))
	}
	depLibraryPaths = appendUnique(depLibraryPaths, filepath.Join(prefix, "lib"))

	completed = true
	if err := s.silo.MarkComplete(name, exactVersion); err != nil {
		return nil, fmt.Errorf("mark complete: %w", err)
	}

	return &AssemblerResult{
		DownloadResults: downloadResults,
		Version:         exactVersion,
		Prefix:          prefix,
		Env: map[string]string{
			"LD_LIBRARY_PATH": strings.Join(depLibraryPaths, ":"),
		},
	}, nil
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

func (s *Service) collectDepFlags(prefix string, cppFlags, ldFlags, pcPaths []string) ([]string, []string, []string) {
	cppFlags = appendUnique(cppFlags, "-I"+filepath.Join(prefix, "include"))
	ldFlags = appendUnique(ldFlags, "-L"+filepath.Join(prefix, "lib"))
	ldFlags = appendUnique(ldFlags, "-Wl,-rpath,"+filepath.Join(prefix, "lib"))
	pcPath := filepath.Join(prefix, "lib", "pkgconfig")
	if _, err := os.Stat(pcPath); err == nil {
		pcPaths = appendUnique(pcPaths, pcPath)
	}
	pc64Path := filepath.Join(prefix, "lib64", "pkgconfig")
	if _, err := os.Stat(pc64Path); err == nil {
		pcPaths = appendUnique(pcPaths, pc64Path)
	}
	return cppFlags, ldFlags, pcPaths
}

// InstallExtension builds a single PHP extension from the PHP source tree
// using phpize. The extension source must be at ext/<name>/ inside the
// PHP source tree. After building, it adds extension=<name>.so to php.ini
// and records the extension in the manifest.
func (s *Service) InstallExtension(phpVersion, extName, phpSourceDir, phpPrefix string) error {
	extDir := filepath.Join(phpSourceDir, "ext", extName)
	if _, err := os.Stat(extDir); os.IsNotExist(err) {
		return fmt.Errorf("extension %q not found in PHP source tree at %s", extName, extDir)
	}

	phpize := filepath.Join(phpPrefix, "bin", "phpize")
	if _, err := os.Stat(phpize); os.IsNotExist(err) {
		return fmt.Errorf("phpize not found at %s (PHP not installed)", phpize)
	}

	phpConfig := filepath.Join(phpPrefix, "bin", "php-config")
	if _, err := os.Stat(phpConfig); os.IsNotExist(err) {
		return fmt.Errorf("php-config not found at %s", phpConfig)
	}

	cmd := exec.Command(phpize)
	cmd.Dir = extDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("phpize %s: %w\n%s", extName, err, out)
	}

	args := []string{"--with-php-config=" + phpConfig}
	extFlags := s.graph.GetExtensionConfigureFlags(extName, phpVersion)
	args = append(args, extFlags...)

	configure := exec.Command("./configure", args...)
	configure.Dir = extDir
	if out, err := configure.CombinedOutput(); err != nil {
		return fmt.Errorf("configure %s: %w\n%s", extName, err, out)
	}

	make := exec.Command("make", "-j4")
	make.Dir = extDir
	if out, err := make.CombinedOutput(); err != nil {
		return fmt.Errorf("make %s: %w\n%s", extName, err, out)
	}

	install := exec.Command("make", "install")
	install.Dir = extDir
	if out, err := install.CombinedOutput(); err != nil {
		return fmt.Errorf("make install %s: %w\n%s", extName, err, out)
	}

	iniDir := filepath.Join(phpPrefix, "etc")
	if err := os.MkdirAll(iniDir, 0755); err != nil {
		return fmt.Errorf("create ini dir: %w", err)
	}
	iniPath := filepath.Join(iniDir, "php.ini")
	entry := "extension=" + extName + ".so\n"
	f, err := os.OpenFile(iniPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open php.ini: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("write php.ini: %w", err)
	}

	manifest, err := s.silo.GetExtensionManifest(phpVersion)
	if err != nil {
		return fmt.Errorf("get extension manifest: %w", err)
	}
	manifest.Extensions = append(manifest.Extensions, domain.ExtensionState{
		Name:        extName,
		InstalledAt: time.Now(),
	})
	if err := s.silo.SaveExtensionManifest(phpVersion, manifest); err != nil {
		return fmt.Errorf("save extension manifest: %w", err)
	}

	return nil
}

// RemoveExtension removes a PHP extension from an installed PHP version.
// It deletes the .so file, removes the extension=<name>.so line from php.ini,
// and updates the extension manifest.
func (s *Service) RemoveExtension(phpVersion, extName, phpPrefix string) error {
	manifest, err := s.silo.GetExtensionManifest(phpVersion)
	if err != nil {
		return fmt.Errorf("get extension manifest: %w", err)
	}

	found := false
	for _, e := range manifest.Extensions {
		if e.Name == extName {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("extension %q is not installed for PHP %s", extName, phpVersion)
	}

	extDir := filepath.Join(phpPrefix, "lib", "php", "extensions")
	removed := false
	filepath.Walk(extDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.Name() == extName+".so" {
			os.Remove(path)
			removed = true
		}
		return nil
	})

	iniPath := filepath.Join(phpPrefix, "etc", "php.ini")
	if data, err := os.ReadFile(iniPath); err == nil {
		lines := strings.Split(string(data), "\n")
		var kept []string
		for _, line := range lines {
			if strings.TrimSpace(line) != "extension="+extName+".so" {
				kept = append(kept, line)
			}
		}
		os.WriteFile(iniPath, []byte(strings.Join(kept, "\n")), 0644)
	}

	var remaining []domain.ExtensionState
	for _, e := range manifest.Extensions {
		if e.Name != extName {
			remaining = append(remaining, e)
		}
	}
	manifest.Extensions = remaining
	if err := s.silo.SaveExtensionManifest(phpVersion, manifest); err != nil {
		return fmt.Errorf("save extension manifest: %w", err)
	}

	if !removed {
		return fmt.Errorf("extension %q .so file not found at %s", extName, extDir)
	}
	return nil
}

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

			regEntry, err := s.reg.Get(n, v)
			if err != nil {
				results[idx].Err = fmt.Errorf("registry resolve %s@%s: %w", n, v, err)
				return
			}
			downloaded, err := s.silo.DownloadURL(regEntry.URL, regEntry.ChecksumType, regEntry.ChecksumValue)
			if err != nil {
				results[idx].Err = fmt.Errorf("download %s@%s: %w", n, v, err)
				return
			}
			results[idx].Downloaded = downloaded

			archivePath := filepath.Join(cacheDir(), filepath.Base(regEntry.URL))
			sourceDir := s.silo.SourcePath(n, v)
			extracted, err := s.silo.Extract(archivePath, sourceDir)
			if err != nil {
				results[idx].Err = fmt.Errorf("extract %s@%s: %w", n, v, err)
				return
			}
			results[idx].Extracted = extracted
		}(i, it.name, it.version)
	}

	wg.Wait()
	for _, dr := range results {
		if dr.Err != nil {
			return results, fmt.Errorf("one or more downloads failed")
		}
	}
	return results, nil
}

func (s *Service) resolveVersion(name, constraint string) (string, error) {
	entries, err := s.reg.List(name)
	if err != nil {
		return "", err
	}
	var versions []string
	for _, e := range entries {
		versions = append(versions, e.Version)
	}
	return resolveVersionConstraint(versions, constraint)
}

func isBuildTool(name string) bool {
	switch name {
	case "m4", "autoconf", "automake", "libtool", "perl", "bison", "re2c", "flex", "zig":
		return true
	}
	return false
}

func isDepInstalled(prefix string) bool {
	info, err := os.Stat(filepath.Join(prefix, "include"))
	if err != nil {
		return false
	}
	return info.IsDir()
}

func cacheDir() string {
	return filepath.Join(resolvePHPVRoot(), "caches")
}

func resolvePHPVRoot(parts ...string) string {
	root := os.Getenv("PHPV_ROOT")
	if root == "" {
		home, _ := os.UserHomeDir()
		root = filepath.Join(home, ".phpv")
	}
	return filepath.Join(append([]string{root}, parts...)...)
}

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

func resolveVersionConstraint(versions []string, constraint string) (string, error) {
	for _, v := range versions {
		if v == constraint {
			return v, nil
		}
	}
	if strings.Count(constraint, ".") == 1 || strings.Count(constraint, ".") == 0 {
		prefix := constraint + "."
		if best := latestMatching(versions, prefix); best != "" {
			return best, nil
		}
	}
	return "", fmt.Errorf("no version matching %q found", constraint)
}

func latestMatching(versions []string, prefix string) string {
	var best string
	for _, v := range versions {
		if strings.HasPrefix(v, prefix) {
			if best == "" || compareVersions(v, best) > 0 {
				best = v
			}
		}
	}
	return best
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

// FindSourceDir locates the actual source directory inside the extracted dir.
func FindSourceDir(extractDir, name, version string) string {
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
