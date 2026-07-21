package assembler

import (
	"context"
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
	"github.com/supanadit/phpv/internal/repository"
	"github.com/supanadit/phpv/internal/repository/memory"
	"github.com/supanadit/phpv/patcher"
	"github.com/supanadit/phpv/registry"
	"github.com/supanadit/phpv/silo"
	"github.com/supanadit/phpv/system"
)

// AssemblerResult holds the outcome of assembling a package.
type AssemblerResult struct {
	DownloadResults  []DownloadResult
	Version          string
	Prefix           string
	Env              map[string]string
	AlreadyInstalled bool
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
// systemPkgs optionally provides a map of available system packages for hybrid builds.
// jobs controls make parallelism (0 = auto).
func (s *Service) Assemble(ctx context.Context, name string, version string, static bool, extensions []string, verbose bool, progress ProgressFunc, systemPkgs map[string]system.Package, jobs int, force bool) (*AssemblerResult, error) {
	emit := func(stage, msg string) {
		if progress != nil {
			progress(stage, msg)
		}
	}

	emit("resolve", fmt.Sprintf("Resolving %s version %q...", name, version))
	exactVersion, err := s.ResolveVersion(name, version)
	if err != nil {
		return nil, fmt.Errorf("resolve version: %w", err)
	}
	emit("resolve", fmt.Sprintf("Resolved %s %s", name, exactVersion))

	prefix := s.silo.PackagePrefix(name, exactVersion)

	state, err := s.silo.GetState(name, exactVersion)
	if err != nil {
		return nil, fmt.Errorf("get state: %w", err)
	}
	if !force && state == domain.StateInstalled {
		return &AssemblerResult{Version: exactVersion, AlreadyInstalled: true}, nil
	}

	if err := s.silo.MarkInProgress(name, exactVersion); err != nil {
		return nil, fmt.Errorf("mark in-progress: %w", err)
	}

	var completed bool
	defer func() {
		if !completed {
			if ctx.Err() != nil {
				s.silo.MarkInterrupted(name, exactVersion)
			} else {
				s.silo.MarkFailed(name, exactVersion)
			}
		}
	}()

	emit("deps", "Resolving dependency graph...")
	plan, err := s.graph.GetBuildPlan(name, exactVersion, extensions)
	if err != nil {
		return nil, fmt.Errorf("resolve build plan for %s@%s: %w", name, exactVersion, err)
	}
	emit("deps", fmt.Sprintf("Found %d dependencies", len(plan.Deps)))
	for _, w := range plan.Warnings {
		emit("deps", w)
	}

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

	// Collect flags from locally-built and system-fallback deps separately so
	// we can place local flags before system flags. This prevents system
	// headers/libraries from shadowing an isolated version when a mixed
	// build plan is used (e.g. libxml2 built from source while curl/openssl
	// come from the system).
	var localCppFlags, localLdFlags, localPcPaths []string
	var systemCppFlags, systemLdFlags, systemPcPaths []string
	var localLibraryPaths []string

	for _, dep := range plan.Deps {
		if isBuildTool(dep.Name) {
			emit("skip", fmt.Sprintf("Skipping build tool %s", dep.Name))
			continue
		}
		if dep.Optional && dep.Version == "" {
			emit("skip", fmt.Sprintf("Skipping optional dep %s (no version specified)", dep.Name))
			continue
		}
		depVersion := extractVersion(dep.Version)
		constraint := extractConstraint(dep.Version)
		depPrefix := s.silo.PackagePrefix(dep.Name, depVersion)
		sourceDir := s.silo.SourcePath(dep.Name, depVersion)

		if isDepInstalled(depPrefix) {
			emit("skip", fmt.Sprintf("Already built %s@%s", dep.Name, depVersion))
			localCppFlags, localLdFlags, localPcPaths = s.collectDepFlags(depPrefix, localCppFlags, localLdFlags, localPcPaths)
			localLibraryPaths = appendUnique(localLibraryPaths, filepath.Join(depPrefix, "lib"))
			continue
		}

		// Hybrid mode: check if system package is available and compatible
		if !static && systemPkgs != nil {
			if sysPkg, ok := systemPkgs[dep.Name]; ok && sysPkg.Installed && sysPkg.Version != "" {
				sysCompat := true
				if constraint != "" {
					sysCompat = repository.MatchVersionRange(constraint, sysPkg.Version)
				}
				if sysCompat {
					emit("system-use", fmt.Sprintf("Using system %s@%s (satisfies %s)", dep.Name, sysPkg.Version, constraint))
					systemCppFlags, systemLdFlags, systemPcPaths = collectSystemFlags(systemCppFlags, systemLdFlags, systemPcPaths)
					continue
				}
				emit("system-incompatible", fmt.Sprintf("System %s@%s doesn't satisfy %s, building from source", dep.Name, sysPkg.Version, constraint))
			}
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
			buildEnv = []string{"CFLAGS=" + strings.Join(prepared.ExtraCFlags, " "), "CXXFLAGS=" + strings.Join(prepared.ExtraCFlags, " ")}
		}
		depFlags := s.graph.GetConfigureFlags(dep.Name, depVersion)
		depFlags = append(depFlags, s.resolveDepPlaceholders(prepared.ConfigureFlags, plan.Deps)...)
		buildDir, _, err := s.forge.Build(ctx, dep.Name, depVersion, sourceDir, buildEnv, depFlags, depPrefix, verbose, jobs)
		if err != nil {
			emit("error", fmt.Sprintf("Build failed for %s@%s", dep.Name, depVersion))
			return nil, fmt.Errorf("forge build %s@%s: %w", dep.Name, depVersion, err)
		}
		emit("install", fmt.Sprintf("Installing %s@%s → %s", dep.Name, depVersion, depPrefix))
		if err := s.forge.Install(ctx, dep.Name, depVersion, buildDir, depPrefix, verbose, jobs); err != nil {
			emit("error", fmt.Sprintf("Install failed for %s@%s", dep.Name, depVersion))
			return nil, fmt.Errorf("forge install %s@%s: %w", dep.Name, depVersion, err)
		}
		emit("done", fmt.Sprintf("✓ %s@%s installed", dep.Name, depVersion))

		localCppFlags, localLdFlags, localPcPaths = s.collectDepFlags(depPrefix, localCppFlags, localLdFlags, localPcPaths)
		localLibraryPaths = appendUnique(localLibraryPaths, filepath.Join(depPrefix, "lib"))
	}

	// Combine flags: local (isolated) flags first so they take precedence over
	// system headers/libraries when both are present.
	depCppFlags := append(localCppFlags, systemCppFlags...)
	depLdFlags := append(localLdFlags, systemLdFlags...)
	depPkgConfigPaths := append(localPcPaths, systemPcPaths...)

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
	if len(plan.CFlags) > 0 || len(plan.CompilerFlags) > 0 {
		allCFlags := plan.CFlags
		allCFlags = append(allCFlags, plan.CompilerFlags...)
		env = setEnvVar(env, "CFLAGS", strings.Join(allCFlags, " "))
		compilerRule := s.graph.GetCompilerStdRule(version)
		cxxflags := memory.CXXFlagsFromCFlagsWithStd(allCFlags, true, compilerRule)
		env = setEnvVar(env, "CXXFLAGS", strings.Join(cxxflags, " "))
	}

	// Isolation: ensure local libs are found before system libs at both
	// link time and runtime. Without this, the system OpenSSL 3.x shadows
	// the locally-built OpenSSL 1.1.1w at link time, causing ABI mismatches
	// and "undefined reference" errors for symbols that differ between versions.
	//
	// 1. Prepend -Wl,-rpath,<local>/lib to LDFLAGS so the linker searches
	//    local lib directories first (before system /usr/lib64).
	// 2. Set LD_LIBRARY_PATH to local lib paths only for runtime isolation.
	// 3. Overwrite CPPFLAGS (not append) so system /usr/include is excluded.
	// 4. Add local include paths to CFLAGS so PHP's configure test programs
	//    find the local OpenSSL headers first (not the system 3.x headers).
	if len(depLdFlags) > 0 {
		var rpathFlags, libPaths []string
		for _, flag := range depLdFlags {
			if strings.HasPrefix(flag, "-L") {
				path := strings.TrimPrefix(flag, "-L")
				rpathFlags = append(rpathFlags, "-Wl,-rpath,"+path)
				libPaths = append(libPaths, path)
			}
		}
		if len(rpathFlags) > 0 {
			env = setEnvVar(env, "LDFLAGS", strings.Join(append(rpathFlags, depLdFlags...), " "))
		}
		if len(libPaths) > 0 {
			env = setEnvVar(env, "LD_LIBRARY_PATH", strings.Join(libPaths, ":"))
		}
	}
	if len(depCppFlags) > 0 {
		// Also add local include paths to CFLAGS so PHP's configure test
		// programs find the local headers first. CPPFLAGS alone is not
		// enough because PHP's configure uses CFLAGS for its test compiles.
		existingCFlags := ""
		for _, e := range env {
			if strings.HasPrefix(e, "CFLAGS=") {
				existingCFlags = strings.TrimPrefix(e, "CFLAGS=")
				break
			}
		}
		if existingCFlags != "" {
			env = setEnvVar(env, "CFLAGS", existingCFlags+" "+strings.Join(depCppFlags, " "))
		} else {
			env = setEnvVar(env, "CFLAGS", strings.Join(depCppFlags, " "))
		}
	}

	configureFlags := plan.ConfigureFlags
	configureFlags = s.resolveDependencyFlags(name, exactVersion, configureFlags, plan.Deps)
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
	buildDir, _, err := s.forge.Build(ctx, name, exactVersion, srcPath, env, configureFlags, prefix, verbose, jobs)
	if err != nil {
		emit("error", fmt.Sprintf("Build failed for %s", name))
		return nil, fmt.Errorf("build %s@%s: %w", name, exactVersion, err)

	}
	emit("install", fmt.Sprintf("Installing %s → %s", name, prefix))
	if err := s.forge.Install(ctx, name, exactVersion, buildDir, prefix, verbose, jobs); err != nil {
		emit("error", fmt.Sprintf("Install failed for %s", name))
		return nil, fmt.Errorf("install %s@%s: %w", name, exactVersion, err)
	}

	emit("done", fmt.Sprintf("%s %s installed at %s", name, exactVersion, prefix))

	localLibraryPaths = appendUnique(localLibraryPaths, filepath.Join(prefix, "lib"))

	completed = true
	if err := s.silo.MarkComplete(name, exactVersion); err != nil {
		return nil, fmt.Errorf("mark complete: %w", err)
	}

	return &AssemblerResult{
		DownloadResults: downloadResults,
		Version:         exactVersion,
		Prefix:          prefix,
		Env: map[string]string{
			"LD_LIBRARY_PATH": strings.Join(localLibraryPaths, ":"),
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

func collectSystemFlags(cppFlags, ldFlags, pcPaths []string) ([]string, []string, []string) {
	cppFlags = appendUnique(cppFlags, "-I/usr/include")
	ldFlags = appendUnique(ldFlags, "-L/usr/lib64")
	ldFlags = appendUnique(ldFlags, "-Wl,-rpath,/usr/lib64")
	pcPaths = appendUnique(pcPaths, "/usr/lib/pkgconfig")
	pcPaths = appendUnique(pcPaths, "/usr/lib64/pkgconfig")
	pcPaths = appendUnique(pcPaths, "/usr/share/pkgconfig")
	return cppFlags, ldFlags, pcPaths
}

// getPHPIniDir returns the directory where PHP expects to find php.ini
// by querying php-config --ini-path. Falls back to prefix/etc if query fails.
func getPHPIniDir(phpPrefix string) string {
	phpConfig := filepath.Join(phpPrefix, "bin", "php-config")
	out, err := exec.Command(phpConfig, "--ini-path").Output()
	if err != nil {
		return filepath.Join(phpPrefix, "etc")
	}
	return strings.TrimSpace(string(out))
}

// InstallExtension builds a single PHP extension from the PHP source tree
// using phpize. The extension source must be at ext/<name>/ inside the
// PHP source tree. After building, it adds extension=<name>.so to php.ini
// and records the extension in the manifest.
func (s *Service) InstallExtension(ctx context.Context, phpVersion, extName, phpSourceDir, phpPrefix string, jobs int) error {
	// Safety check: skip if the extension is already built into PHP.
	// This prevents "Module already loaded" warnings for extensions that
	// are compiled into the PHP binary (e.g. iconv in PHP 8.5+).
	phpBin := filepath.Join(phpPrefix, "bin", "php")
	if fi, err := os.Stat(phpBin); err == nil && !fi.IsDir() {
		cmd := exec.Command(phpBin, "-m")
		out, err := cmd.Output()
		if err == nil {
			modules := strings.Split(string(out), "\n")
			for _, mod := range modules {
				mod = strings.TrimSpace(mod)
				if strings.EqualFold(mod, extName) {
					return nil
				}
			}
		}
	}

	extDir := filepath.Join(phpSourceDir, "ext", extName)
	if _, err := os.Stat(extDir); os.IsNotExist(err) {
		return fmt.Errorf("extension %q not found in PHP source tree at %s", extName, extDir)
	}

	configM4 := filepath.Join(extDir, "config.m4")
	config0M4 := filepath.Join(extDir, "config0.m4")
	if _, err := os.Stat(configM4); os.IsNotExist(err) {
		if _, err := os.Stat(config0M4); err == nil {
			if err := os.Rename(config0M4, configM4); err != nil {
				return fmt.Errorf("restore config.m4 for %s: %w", extName, err)
			}
		}
	}

	phpize := filepath.Join(phpPrefix, "bin", "phpize")
	if _, err := os.Stat(phpize); os.IsNotExist(err) {
		return fmt.Errorf("phpize not found at %s (PHP not installed)", phpize)
	}

	phpConfig := filepath.Join(phpPrefix, "bin", "php-config")
	if _, err := os.Stat(phpConfig); os.IsNotExist(err) {
		return fmt.Errorf("php-config not found at %s", phpConfig)
	}

	cmd := exec.CommandContext(ctx, phpize)
	cmd.Dir = extDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("phpize %s: %w\n%s", extName, err, out)
	}

	args := []string{"--with-php-config=" + phpConfig}
	extFlags := s.graph.GetExtensionConfigureFlags(extName, phpVersion)
	args = append(args, extFlags...)

	configure := exec.CommandContext(ctx, "./configure", args...)
	configure.Dir = extDir
	if out, err := configure.CombinedOutput(); err != nil {
		return fmt.Errorf("configure %s: %w\n%s", extName, err, out)
	}

	make := exec.CommandContext(ctx, "make", fmt.Sprintf("-j%d", jobs))
	make.Dir = extDir
	if out, err := make.CombinedOutput(); err != nil {
		return fmt.Errorf("make %s: %w\n%s", extName, err, out)
	}

	install := exec.CommandContext(ctx, "make", "install")
	install.Dir = extDir
	if out, err := install.CombinedOutput(); err != nil {
		return fmt.Errorf("make install %s: %w\n%s", extName, err, out)
	}

	iniDir := getPHPIniDir(phpPrefix)
	if err := os.MkdirAll(iniDir, 0755); err != nil {
		return fmt.Errorf("create ini dir: %w", err)
	}
	iniPath := filepath.Join(iniDir, "php.ini")
	entry := "extension=" + extName + ".so"
	// Check if the extension line already exists in php.ini to avoid duplicates.
	if data, err := os.ReadFile(iniPath); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.TrimSpace(line) == entry {
				return nil
			}
		}
	}
	f, err := os.OpenFile(iniPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open php.ini: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(entry + "\n"); err != nil {
		return fmt.Errorf("write php.ini: %w", err)
	}

	manifest, err := s.silo.GetExtensionManifest(phpVersion)
	if err != nil {
		return fmt.Errorf("get extension manifest: %w", err)
	}
	// Deduplicate manifest: skip if this extension is already recorded.
	for _, e := range manifest.Extensions {
		if e.Name == extName {
			return nil
		}
	}
	manifest.Extensions = append(manifest.Extensions, domain.ExtensionState{
		Name:        extName,
		Type:        domain.ExtensionTypeBuiltin,
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

	iniPath := filepath.Join(getPHPIniDir(phpPrefix), "php.ini")
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
		if dep.Optional && dep.Version == "" {
			continue
		}
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
	var failed []string
	for _, dr := range results {
		if dr.Err != nil {
			failed = append(failed, fmt.Sprintf("%s@%s: %v", dr.Name, dr.Version, dr.Err))
		}
	}
	if len(failed) > 0 {
		return results, fmt.Errorf("download failed:\n  - %s", strings.Join(failed, "\n  - "))
	}
	return results, nil
}

func (s *Service) ResolveVersion(name, constraint string) (string, error) {
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

func extractConstraint(v string) string {
	if v == "" {
		return ""
	}
	if _, after, found := strings.Cut(v, "|"); found {
		return after
	}
	return ""
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
		if an > bn {
			return 1
		}
		if an < bn {
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
	// ICU tarballs extract to icu/source/ (nested one level deeper).
	if name == "icu" {
		icuSource := filepath.Join(extractDir, "icu", "source")
		if hasBuildFile(icuSource) {
			return icuSource
		}
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
