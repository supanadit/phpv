package bundle

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/supanadit/phpv/assembler"
	"github.com/supanadit/phpv/domain"
)

// PublishOpts controls the portable bundle publishing process.
type PublishOpts struct {
	Version    string
	OutputPath string
	Jobs       int
	Force      bool
}

// Publish builds a portable musl-static PHP bundle from source.
// It builds PHP core with --static, builds the 25 default extensions as shared,
// captures the PHP API version, and emits a v2 bundle with ExtPool + Toolchain.
func (s *Service) Publish(ctx context.Context, opts PublishOpts) error {
	version := opts.Version
	fmt.Printf("Building portable PHP %s (musl-static)...\n", version)

	// Step 1: Build PHP core with --static (no system deps, no extensions).
	defaultExts, _ := s.graph.DefaultExtensions(version)
	fmt.Printf("Default extensions: %d\n", len(defaultExts))

	result, err := s.assembler.Assemble(ctx, "php", version, true, nil, true, nil, nil, opts.Jobs, opts.Force)
	if err != nil {
		return fmt.Errorf("build PHP core: %w", err)
	}
	prefix := result.Prefix
	exactVersion := result.Version

	// Step 2: Build each default extension as shared.
	srcDir := s.silo.SourcePath("php", exactVersion)
	srcPath := assembler.FindSourceDir(srcDir, "php", exactVersion)
	if srcPath == "" {
		return fmt.Errorf("PHP source not found at %s", srcDir)
	}

	sharedOnly := s.graph.SharedOnlyExtensions(exactVersion, defaultExts)
	var extPool []domain.BundleExtArtifact

	for _, ext := range defaultExts {
		fmt.Printf("  Building extension %s...\n", ext)
		if err := s.assembler.InstallExtension(ctx, exactVersion, ext, srcPath, prefix, opts.Jobs); err != nil {
			fmt.Printf("  Warning: %s build failed: %v\n", ext, err)
			continue
		}
	}

	// Step 3: Build shared-only extensions (those that can't be built statically).
	for _, ext := range sharedOnly {
		fmt.Printf("  Building shared extension %s...\n", ext)
		if err := s.assembler.InstallExtension(ctx, exactVersion, ext, srcPath, prefix, opts.Jobs); err != nil {
			fmt.Printf("  Warning: %s shared build failed: %v\n", ext, err)
			continue
		}
	}

	// Step 4: Collect pre-built .so files into the ExtPool.
	extManifest, err := s.silo.GetExtensionManifest(exactVersion)
	if err != nil {
		return fmt.Errorf("get extension manifest: %w", err)
	}

	phpAPI := getPhpAPIVersion(prefix)
	extsDir := filepath.Join(prefix, "exts")
	if err := os.MkdirAll(extsDir, 0755); err != nil {
		return fmt.Errorf("create exts dir: %w", err)
	}

	for _, ext := range extManifest.Extensions {
		soName := ext.Name + ".so"
		// Find the .so that was installed by make install.
		extDir := getExtensionDir(prefix)
		srcSO := filepath.Join(extDir, soName)
		if _, err := os.Stat(srcSO); os.IsNotExist(err) {
			continue
		}
		dstSO := filepath.Join(extsDir, soName)
		if err := copyFile(srcSO, dstSO); err != nil {
			return fmt.Errorf("copy %s .so: %w", ext.Name, err)
		}
		extPool = append(extPool, domain.BundleExtArtifact{
			Name:          ext.Name,
			Version:       ext.Version,
			SOFile:        soName,
			PhpApiVersion: phpAPI,
		})
	}

	// Step 5: Strip binaries.
	stripBinaries(prefix)

	// Step 6: Build the manifest.
	manifest := domain.BundleManifest{
		FormatVersion: 2,
		Package:       "php",
		Version:       exactVersion,
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		Libc:          "musl",
		PhpApiVersion: phpAPI,
		BuildDate:     time.Now(),
		Builder: domain.BundleBuilder{
			PHPVVersion: "1.0.0",
			Compiler:    "gcc",
			Static:      true,
			Libc:        "musl",
		},
		ExtPool:   extPool,
		Toolchain: domain.BundleToolchain{},
		TotalSize: 0,
	}

	outputPath := opts.OutputPath
	if outputPath == "" {
		outputPath = fmt.Sprintf("php-%s-linux-%s-musl.tar.gz", exactVersion, runtime.GOARCH)
	}

	fmt.Printf("Creating bundle %s...\n", outputPath)
	if err := exportBundle(manifest, prefix, outputPath); err != nil {
		return fmt.Errorf("create bundle: %w", err)
	}

	fmt.Printf("✓ Portable PHP %s bundle created: %s\n", exactVersion, outputPath)
	fmt.Printf("  Extensions: %d pre-built .so files\n", len(extPool))
	fmt.Printf("  PHP API: %s\n", phpAPI)
	return nil
}

// getPhpAPIVersion returns the PHP Module API version from php-config.
func getPhpAPIVersion(phpPrefix string) string {
	phpConfig := filepath.Join(phpPrefix, "bin", "php-config")
	out, err := exec.Command(phpConfig, "--extension-dir").Output()
	if err != nil {
		return ""
	}
	dir := filepath.ToSlash(strings.TrimSpace(string(out)))
	dir = strings.TrimSuffix(dir, "/")
	parts := strings.Split(dir, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	return ""
}

// getExtensionDir returns the PHP extension directory from php-config.
func getExtensionDir(phpPrefix string) string {
	phpConfig := filepath.Join(phpPrefix, "bin", "php-config")
	out, err := exec.Command(phpConfig, "--extension-dir").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// stripBinaries strips debug symbols from PHP binaries and .so files.
func stripBinaries(prefix string) {
	paths := []string{
		filepath.Join(prefix, "bin", "php"),
		filepath.Join(prefix, "bin", "php-cgi"),
		filepath.Join(prefix, "bin", "phpdbg"),
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			exec.Command("strip", p).Run()
		}
	}
	// Strip all .so files in the extensions directory.
	extDir := getExtensionDir(prefix)
	if extDir != "" {
		filepath.Walk(extDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if filepath.Ext(path) == ".so" {
				exec.Command("strip", path).Run()
			}
			return nil
		})
	}
	// Also strip .so files in the exts/ pool.
	extsDir := filepath.Join(prefix, "exts")
	if _, err := os.Stat(extsDir); err == nil {
		filepath.Walk(extsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if filepath.Ext(path) == ".so" {
				exec.Command("strip", path).Run()
			}
			return nil
		})
	}
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
