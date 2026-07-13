package disk

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/supanadit/phpv/assembler"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/forge"
	"github.com/supanadit/phpv/internal/repository/memory"
	"github.com/supanadit/phpv/patcher"
	"github.com/supanadit/phpv/registry"
	"github.com/supanadit/phpv/silo"
)

type AssemblerRepository struct {
	silo    silo.SiloRepository
	forge   *forge.Service
	patcher *patcher.Service
	reg     registry.RegistryRepository
	graph   *memory.AssemblerRepository
}

func NewAssemblerRepository(sr silo.SiloRepository, fr *forge.Service, pr *patcher.Service, rr registry.RegistryRepository, gr *memory.AssemblerRepository) *AssemblerRepository {
	return &AssemblerRepository{
		silo:    sr,
		forge:   fr,
		patcher: pr,
		reg:     rr,
		graph:   gr,
	}
}

func (r *AssemblerRepository) GetOrderedDependencies(name string, version string) ([]domain.Dependency, error) {
	return r.graph.GetOrderedDependencies(name, version)
}

func (r *AssemblerRepository) Assemble(name string, version string, progress assembler.ProgressFunc) (*assembler.AssemblerResult, error) {
	emit := func(stage, msg string) {
		if progress != nil {
			progress(stage, msg)
		}
	}

	emit("resolve", fmt.Sprintf("Resolving php version %q...", version))
	exactVersion, err := r.resolveVersion(name, version)
	if err != nil {
		return nil, fmt.Errorf("resolve version: %w", err)
	}
	emit("resolve", fmt.Sprintf("Resolved php %s", exactVersion))

	state, err := r.silo.GetState(exactVersion)
	if err != nil {
		return nil, fmt.Errorf("get state: %w", err)
	}
	if state == domain.StateInstalled {
		emit("done", fmt.Sprintf("PHP %s is already installed", exactVersion))
		return &assembler.AssemblerResult{PHPVersion: exactVersion}, nil
	}

	if err := r.silo.MarkInProgress(exactVersion); err != nil {
		return nil, fmt.Errorf("mark in-progress: %w", err)
	}

	var completed bool
	defer func() {
		if !completed {
			r.silo.MarkFailed(exactVersion)
		}
	}()

	emit("deps", "Resolving dependency graph...")
	deps, err := r.graph.GetOrderedDependencies(name, exactVersion)
	if err != nil {
		return nil, fmt.Errorf("resolve deps for %s@%s: %w", name, exactVersion, err)
	}
	emit("deps", fmt.Sprintf("Found %d dependencies", len(deps)))

	emit("download", fmt.Sprintf("Downloading and extracting %d packages...", len(deps)+1))
	downloadResults, err := r.downloadAll(name, exactVersion, deps)
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
	for _, dep := range deps {
		if isBuildTool(dep.Name) {
			emit("skip", fmt.Sprintf("Skipping build tool %s", dep.Name))
			continue
		}
		depVersion := extractVersion(dep.Version)
		prefix := r.silo.DependencyPath(exactVersion, dep.Name, depVersion)
		sourceDir := r.silo.SourcePath(dep.Name, depVersion)

		if isDepInstalled(prefix) {
			emit("skip", fmt.Sprintf("Already built %s@%s", dep.Name, depVersion))
			depCppFlags, depLdFlags, depPkgConfigPaths = r.collectDepFlags(prefix, depCppFlags, depLdFlags, depPkgConfigPaths)
			continue
		}

		emit("build", fmt.Sprintf("Building %s@%s...", dep.Name, depVersion))
		if err := r.applyPatches(dep.Name, depVersion, sourceDir, emit); err != nil {
			emit("error", fmt.Sprintf("Patch failed for %s@%s: %v", dep.Name, depVersion, err))
			return nil, err
		}
		var buildEnv []string
		if flags := r.extraCFlags(dep.Name, depVersion); len(flags) > 0 {
			buildEnv = []string{"CFLAGS=" + strings.Join(flags, " ")}
		}
		extraFlags := r.extraConfigureFlags(dep.Name, depVersion, exactVersion, prefix, sourceDir)
		buildDir, _, err := r.forge.Build(dep.Name, depVersion, sourceDir, buildEnv, extraFlags, prefix)
		if err != nil {
			emit("error", fmt.Sprintf("Build failed for %s@%s", dep.Name, depVersion))
			return nil, fmt.Errorf("forge build %s@%s: %w", dep.Name, depVersion, err)
		}
		emit("install", fmt.Sprintf("Installing %s@%s → %s", dep.Name, depVersion, prefix))
		if err := r.forge.Install(dep.Name, depVersion, buildDir, prefix); err != nil {
			emit("error", fmt.Sprintf("Install failed for %s@%s", dep.Name, depVersion))
			return nil, fmt.Errorf("forge install %s@%s: %w", dep.Name, depVersion, err)
		}
		emit("done", fmt.Sprintf("✓ %s@%s installed", dep.Name, depVersion))

		depCppFlags, depLdFlags, depPkgConfigPaths = r.collectDepFlags(prefix, depCppFlags, depLdFlags, depPkgConfigPaths)
	}

	phpPrefix := r.silo.PHPOutputPath(exactVersion)
	phpSourceDir := r.silo.SourcePath(name, exactVersion)
	phpSrcPath := findSourceDir(phpSourceDir, name, exactVersion)
	if phpSrcPath == "" {
		return nil, fmt.Errorf("could not find source directory in %s", phpSourceDir)
	}

	configureFlags := []string{
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
	if err := r.applyPatches(name, exactVersion, phpSrcPath, emit); err != nil {
		emit("error", fmt.Sprintf("Patch failed for %s: %v", name, err))
		return nil, err
	}

	emit("make", fmt.Sprintf("Compiling %s (this may take a while)...", name))
	buildDir, _, err := r.forge.Build("php", exactVersion, phpSrcPath, env, configureFlags, phpPrefix)
	if err != nil {
		emit("error", fmt.Sprintf("Build failed for %s", name))
		return nil, fmt.Errorf("build php %s@%s: %w", name, exactVersion, err)
	}
	if err := r.forge.Install("php", exactVersion, buildDir, phpPrefix); err != nil {
		emit("error", fmt.Sprintf("Install failed for %s", name))
		return nil, fmt.Errorf("install php %s@%s: %w", name, exactVersion, err)
	}

	emit("done", fmt.Sprintf("✓ %s %s installed at %s", name, exactVersion, phpPrefix))

	var depLibraryPaths []string
	for _, dep := range deps {
		if isBuildTool(dep.Name) {
			continue
		}
		depVersion := extractVersion(dep.Version)
		depLibraryPaths = appendUnique(depLibraryPaths, filepath.Join(r.silo.DependencyPath(exactVersion, dep.Name, depVersion), "lib"))
	}
	depLibraryPaths = appendUnique(depLibraryPaths, filepath.Join(phpPrefix, "lib"))

	completed = true
	if err := r.silo.MarkComplete(exactVersion); err != nil {
		return nil, fmt.Errorf("mark complete: %w", err)
	}

	return &assembler.AssemblerResult{
		DownloadResults: downloadResults,
		PHPVersion:      exactVersion,
		Prefix:          phpPrefix,
		Env: map[string]string{
			"LD_LIBRARY_PATH": strings.Join(depLibraryPaths, ":"),
		},
	}, nil
}

func (r *AssemblerRepository) collectDepFlags(prefix string, cppFlags, ldFlags, pcPaths []string) ([]string, []string, []string) {
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

func (r *AssemblerRepository) applyPatches(name, version, sourceDir string, emit assembler.ProgressFunc) error {
	patches := r.patcher.PatchesFor(name, version)
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

func (r *AssemblerRepository) extraCFlags(name, version string) []string {
	patches := r.patcher.PatchesFor(name, version)
	for _, p := range patches {
		if p.ExtraCFlags != nil {
			return p.ExtraCFlags
		}
	}
	return nil
}

func (r *AssemblerRepository) extraConfigureFlags(name, depVersion, phpVersion, prefix, sourceDir string) []string {
	patches := r.patcher.PatchesFor(name, depVersion)
	for _, p := range patches {
		if len(p.ConfigureFlags) == 0 {
			continue
		}
		resolved := make([]string, 0, len(p.ConfigureFlags))
		for _, flag := range p.ConfigureFlags {
			flag = strings.ReplaceAll(flag, "{{prefix}}", prefix)
			flag = strings.ReplaceAll(flag, "{{source}}", sourceDir)
			if strings.Contains(flag, "{{dep:") {
				start := strings.Index(flag, "{{dep:")
				end := strings.Index(flag[start:], "}}")
				if end > 0 {
					depName := flag[start+6 : start+end]
					depBase := filepath.Dir(r.silo.DependencyPath(phpVersion, depName, "1.0.0"))
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

func (r *AssemblerRepository) downloadAll(name, version string, deps []domain.Dependency) ([]assembler.DownloadResult, error) {
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

	results := make([]assembler.DownloadResult, len(items))
	var wg sync.WaitGroup

	for i, it := range items {
		wg.Add(1)
		go func(idx int, n, v string) {
			defer wg.Done()
			results[idx] = assembler.DownloadResult{Name: n, Version: v}

			regEntry, err := r.reg.Get(n, v, false, runtime.GOOS)
			if err != nil {
				results[idx].Err = fmt.Errorf("registry resolve %s@%s: %w", n, v, err)
				return
			}
			downloaded, err := r.silo.Download(regEntry.URL, regEntry.ChecksumType, regEntry.ChecksumValue)
			if err != nil {
				results[idx].Err = fmt.Errorf("download %s@%s: %w", n, v, err)
				return
			}
			results[idx].Downloaded = downloaded

			archivePath := filepath.Join(cacheDir(), filepath.Base(regEntry.URL))
			sourceDir := r.silo.SourcePath(n, v)
			extracted, err := r.silo.Extract(archivePath, sourceDir)
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

func (r *AssemblerRepository) resolveVersion(name, constraint string) (string, error) {
	entries, err := r.reg.List(name, false, "")
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
