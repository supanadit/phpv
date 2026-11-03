package assembler

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/supanadit/phpv/compiler"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/internal/source"
)

// ServerPackageName maps the user-facing server name (domain.ServerCaddy,
// domain.ServerNginx, domain.ServerApache) to its registry package name.
// Apache is known to users as "apache" but its upstream project and archive
// are "httpd".
func ServerPackageName(server string) string {
	if server == domain.ServerApache {
		return "httpd"
	}
	return server
}

// AssembleServer installs a webserver package (caddy, nginx, httpd) into its
// silo prefix. It reuses the regular pipeline: resolve → download → build
// deps → configure/make/install, with server-specific handling:
//
//   - caddy ships a prebuilt static binary: no forge build, just copy the
//     binary out of the extracted archive into <prefix>/bin;
//   - nginx always builds pcre2 and zlib from source so the generated
//     configuration never depends on system library variance;
//   - httpd vendors apr/apr-util into srclib/ at configure time
//     (--with-included-apr), so those deps are downloaded and copied but not
//     built standalone.
//
// Returns the install prefix. Already-installed servers short-circuit.
func (s *Service) AssembleServer(ctx context.Context, name, version string, verbose bool, jobs int, tc compiler.Toolchain) (string, error) {
	emit := func(stage, msg string) {
		if verbose {
			fmt.Printf("[%s] %s\n", stage, msg)
		}
	}

	pkgName := ServerPackageName(name)
	exact, err := s.ResolveVersion(pkgName, version)
	if err != nil {
		return "", fmt.Errorf("resolve %s version: %w", name, err)
	}
	prefix := s.silo.PackagePrefix(name, exact)
	if isDepInstalled(prefix) {
		emit("skip", fmt.Sprintf("Already installed %s@%s", name, exact))
		return prefix, nil
	}
	if jobs < 1 {
		jobs = 4
	}

	if err := s.silo.MarkInProgress(name, exact); err != nil {
		return "", fmt.Errorf("mark in-progress: %w", err)
	}
	var completed bool
	defer func() {
		if !completed {
			if ctx.Err() != nil {
				s.silo.MarkInterrupted(name, exact)
			} else {
				s.silo.MarkFailed(name, exact)
			}
		}
	}()

	plan, err := s.graph.GetBuildPlan(pkgName, exact, nil, tc.Name)
	if err != nil {
		return "", fmt.Errorf("resolve build plan for %s@%s: %w", name, exact, err)
	}

	mustBuild := map[string]bool{}
	if name == "nginx" {
		// Deterministic nginx builds: never mix system pcre2/zlib.
		mustBuild["pcre2"] = true
		mustBuild["zlib"] = true
	}

	emit("download", fmt.Sprintf("Downloading %s %s...", pkgName, exact))
	if _, err := s.downloadAll(pkgName, exact, plan.Deps, mustBuild, nil, false, emit); err != nil {
		return "", err
	}

	switch pkgName {
	case "caddy":
		if err := s.installCaddyBinary(pkgName, exact, prefix); err != nil {
			return "", err
		}
	case "nginx":
		// nginx compiles pcre2 and zlib from their source trees itself
		// (--with-pcre=<src>, --with-zlib=<src>), so no standalone dep
		// builds — only download/extract, then configure nginx.
		if err := s.buildServerPackage(ctx, pkgName, exact, plan.Deps, prefix, verbose, jobs, tc); err != nil {
			return "", err
		}
	case "httpd":
		// apr/apr-util are vendored into the httpd source tree, not built
		// standalone. pcre2 IS built standalone (httpd invokes pcre2-config
		// from the install prefix). Filter apr/apr-util out of the dep
		// build list, build the rest, vendor the apr sources, then build.
		var httpdDeps []domain.Dependency
		for _, dep := range plan.Deps {
			if dep.Name == "apr" || dep.Name == "apr-util" {
				continue
			}
			httpdDeps = append(httpdDeps, dep)
		}
		if _, err := s.buildDependencies(ctx, httpdDeps, mustBuild, nil, false, verbose, jobs, tc, emit); err != nil {
			return "", err
		}
		aprVers := planDepVersions(plan.Deps, "apr", "apr-util")
		if err := s.vendorAPR(exact, aprVers[0], aprVers[1], emit); err != nil {
			return "", err
		}
		if err := s.buildServerPackage(ctx, pkgName, exact, httpdDeps, prefix, verbose, jobs, tc); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unsupported server package: %s", name)
	}

	completed = true
	if err := s.silo.MarkComplete(name, exact); err != nil {
		return "", fmt.Errorf("mark complete: %w", err)
	}
	emit("done", fmt.Sprintf("%s %s installed at %s", name, exact, prefix))
	return prefix, nil
}

// buildServerPackage runs configure/make/install for a source-built server.
func (s *Service) buildServerPackage(ctx context.Context, name, version string, deps []domain.Dependency, prefix string, verbose bool, jobs int, tc compiler.Toolchain) error {
	emit := func(stage, msg string) {
		if verbose {
			fmt.Printf("[%s] %s\n", stage, msg)
		}
	}
	sourceDir := s.silo.SourcePath(name, version)
	srcPath := source.Find(sourceDir, name)
	if srcPath == "" {
		return fmt.Errorf("could not find source directory in %s", sourceDir)
	}

	flags := s.graph.GetConfigureFlags(name, version)
	flags = s.resolveDepPlaceholders(flags, deps)

	buildDir, _, err := s.forge.Build(ctx, name, version, srcPath, tc.Env(), flags, prefix, verbose, jobs)
	if err != nil {
		return fmt.Errorf("build %s@%s: %w", name, version, err)
	}
	emit("install", fmt.Sprintf("Installing %s@%s → %s", name, version, prefix))
	if err := s.forge.Install(ctx, name, version, buildDir, prefix, verbose, jobs); err != nil {
		return fmt.Errorf("install %s@%s: %w", name, version, err)
	}
	return nil
}

// installCaddyBinary extracts the prebuilt caddy binary from the downloaded
// archive into <prefix>/bin/caddy. Caddy ships no build files, so the plain
// extract dir is scanned instead of source.Find (which requires configure/
// Makefile markers).
func (s *Service) installCaddyBinary(name, version, prefix string) error {
	srcPath := s.silo.SourcePath(name, version)
	if fi, err := os.Stat(srcPath); err != nil || !fi.IsDir() {
		return fmt.Errorf("caddy archive not extracted at %s", srcPath)
	}
	bin := FindServerBinary(srcPath, "caddy")
	if bin == "" {
		return fmt.Errorf("caddy binary not found in extracted archive at %s", srcPath)
	}
	if err := os.MkdirAll(filepath.Join(prefix, "bin"), 0o755); err != nil {
		return fmt.Errorf("create prefix: %w", err)
	}
	dst := filepath.Join(prefix, "bin", "caddy")
	if err := copyFileMode(bin, dst, 0o755); err != nil {
		return fmt.Errorf("install caddy binary: %w", err)
	}
	return nil
}

// vendorAPR copies the downloaded apr/apr-util source trees into the httpd
// source tree as srclib/apr and srclib/apr-util, as required by
// --with-included-apr.
func (s *Service) vendorAPR(httpdVersion string, aprVersion, aprUtilVersion string, emit func(stage, msg string)) error {
	httpdSrc := source.Find(s.silo.SourcePath("httpd", httpdVersion), "httpd")
	if httpdSrc == "" {
		return fmt.Errorf("httpd source not found under %s", s.silo.SourcePath("httpd", httpdVersion))
	}
	pairs := []struct{ pkg, ver, dest string }{
		{"apr", aprVersion, "srclib/apr"},
		{"apr-util", aprUtilVersion, "srclib/apr-util"},
	}
	for _, p := range pairs {
		src := source.Find(s.silo.SourcePath(p.pkg, p.ver), p.pkg)
		if src == "" {
			return fmt.Errorf("%s %s source not found (httpd build needs it vendored)", p.pkg, p.ver)
		}
		dst := filepath.Join(httpdSrc, p.dest)
		if _, err := os.Stat(filepath.Join(dst, "configure")); err == nil {
			continue // already vendored
		}
		emit("prepare", fmt.Sprintf("Vendoring %s → %s", p.pkg, p.dest))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := copyTree(src, dst); err != nil {
			return fmt.Errorf("vendor %s: %w", p.pkg, err)
		}
	}
	return nil
}

// planDepVersions returns the exact pinned versions of the named deps from
// a build plan's dependency list ("" for names not present).
func planDepVersions(deps []domain.Dependency, names ...string) []string {
	out := make([]string, len(names))
	for i, n := range names {
		for _, d := range deps {
			if d.Name != n {
				continue
			}
			v, _ := domain.SplitVersionSpec(d.Version)
			out[i] = v
		}
	}
	return out
}

// FindServerBinary locates an executable inside an extracted archive,
// checking the root, bin/ and a single nested top-level directory.
func FindServerBinary(dir, name string) string {
	candidates := []string{
		filepath.Join(dir, name),
		filepath.Join(dir, "bin", name),
		filepath.Join(dir, "sbin", name),
	}
	entries, err := os.ReadDir(dir)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				candidates = append(candidates,
					filepath.Join(dir, e.Name(), name),
					filepath.Join(dir, e.Name(), "bin", name),
				)
			}
		}
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() && fi.Mode()&0o111 != 0 {
			return c
		}
	}
	return ""
}

func copyFileMode(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// latestLocalVersion returns the newest installed version directory name for
// a package under packages/<name>/, or "" when none exist.
func latestLocalVersion(pkgDir string) string {
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		return ""
	}
	var best string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if best == "" || domain.CompareVersions(e.Name(), best) > 0 {
			best = e.Name()
		}
	}
	return best
}
