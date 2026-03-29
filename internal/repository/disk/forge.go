package disk

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/afero"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/download"
	"github.com/supanadit/phpv/source"
	"github.com/supanadit/phpv/unload"
)

type ForgeRepository struct {
	downloadRepo download.DownloadRepository
	unloadRepo   unload.UnloadRepository
	siloRepo     *SiloRepository
	sourceRepo   source.SourceRepository
	fs           afero.Fs
}

func NewForgeRepository(downloadRepo download.DownloadRepository, unloadRepo unload.UnloadRepository, siloRepo *SiloRepository, sourceRepo source.SourceRepository) *ForgeRepository {
	return &ForgeRepository{
		downloadRepo: downloadRepo,
		unloadRepo:   unloadRepo,
		siloRepo:     siloRepo,
		sourceRepo:   sourceRepo,
		fs:           afero.NewOsFs(),
	}
}

func (r *ForgeRepository) Build(config domain.ForgeConfig) (domain.Forge, error) {
	strategy := r.detectStrategy(config.Name, config.Version)
	return r.BuildWithStrategy(config, strategy)
}

func (r *ForgeRepository) detectStrategy(name, version string) domain.BuildStrategy {
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

func (r *ForgeRepository) BuildWithStrategy(config domain.ForgeConfig, strategy domain.BuildStrategy) (domain.Forge, error) {
	url, err := r.resolveURL(config.Name, config.Version)
	if err != nil {
		return domain.Forge{}, err
	}

	if err := r.ensureSource(config.Name, config.Version, url); err != nil {
		return domain.Forge{}, err
	}

	silo, err := r.siloRepo.GetSilo()
	if err != nil {
		return domain.Forge{}, err
	}

	sourceDir := silo.GetSourceDirPath(config.Name, config.Version)
	installDir := config.Prefix
	if installDir == "" {
		installDir = silo.GetVersionPath(config.Name, config.Version)
	}

	r.ensureFs()

	r.chmodBuildScripts(sourceDir)

	env := r.buildEnv(config)

	switch strategy {
	case domain.StrategyCMake:
		return r.buildCMake(sourceDir, installDir, config, env)
	case domain.StrategyMakeOnly:
		return r.buildMakeOnly(sourceDir, installDir, config, env)
	case domain.StrategyConfigureMake:
		return r.buildConfigureMake(sourceDir, installDir, config, env)
	case domain.StrategyAutogen:
		return r.buildAutogen(sourceDir, installDir, config, env)
	default:
		return domain.Forge{}, fmt.Errorf("unsupported build strategy: %s", strategy)
	}
}

func (r *ForgeRepository) resolveURL(name, version string) (string, error) {
	sourceSvc := source.NewService(r.sourceRepo)

	sources, err := sourceSvc.GetVersions()
	if err != nil {
		return "", err
	}

	for _, src := range sources {
		if src.Name == name && src.Version == version {
			return src.URL, nil
		}
	}

	return "", fmt.Errorf("source not found for %s version %s", name, version)
}

func (r *ForgeRepository) ensureSource(name, version, url string) error {
	silo, err := r.siloRepo.GetSilo()
	if err != nil {
		return err
	}

	cachePath := silo.GetArchivePath(name, version)

	cacheExists, _ := afero.Exists(r.fs, cachePath)
	if !cacheExists {
		cacheDir := filepath.Dir(cachePath)
		if err := r.fs.MkdirAll(cacheDir, 0o755); err != nil {
			return fmt.Errorf("failed to create cache directory: %w", err)
		}

		downloadSvc := download.NewService(r.downloadRepo)
		if _, err := downloadSvc.Download(url, cachePath); err != nil {
			return fmt.Errorf("failed to download %s: %w", url, err)
		}
		fmt.Println("Downloaded:", cachePath)
	} else {
		fmt.Println("Using cached:", cachePath)
	}

	sourceDir := silo.GetSourceDirPath(name, version)
	sourceExists, _ := afero.Exists(r.fs, sourceDir)
	if !sourceExists {
		sourceBaseDir := filepath.Dir(sourceDir)
		if err := r.fs.MkdirAll(sourceBaseDir, 0o755); err != nil {
			return fmt.Errorf("failed to create source directory: %w", err)
		}

		unloadSvc := unload.NewService(r.unloadRepo)
		if _, err := unloadSvc.Unpack(cachePath, sourceDir); err != nil {
			return fmt.Errorf("failed to extract %s: %w", cachePath, err)
		}

		entries, _ := afero.ReadDir(r.fs, sourceDir)
		if len(entries) == 1 && entries[0].IsDir() {
			extractedFolder := filepath.Join(sourceDir, entries[0].Name())
			extractedEntries, _ := afero.ReadDir(r.fs, extractedFolder)
			for _, f := range extractedEntries {
				src := filepath.Join(extractedFolder, f.Name())
				dst := filepath.Join(sourceDir, f.Name())
				if err := r.fs.Rename(src, dst); err != nil {
					return fmt.Errorf("failed to move extracted files: %w", err)
				}
			}
			r.fs.Remove(extractedFolder)
		}
		fmt.Printf("Extracted to: %s\n", sourceDir)
	} else {
		fmt.Println("Using cached source:", sourceDir)
	}

	return nil
}

func (r *ForgeRepository) ensureFs() {
	if r.fs == nil {
		r.fs = afero.NewOsFs()
	}
}

func (r *ForgeRepository) buildEnv(config domain.ForgeConfig) []string {
	env := os.Environ()

	buildToolsPath := filepath.Join(r.siloRepo.silo.Root, "build-tools")
	buildToolsBinPath := r.buildToolsBinPath(buildToolsPath)

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

func (r *ForgeRepository) buildToolsBinPath(buildToolsPath string) string {
	var binPaths []string

	entries, err := afero.ReadDir(r.fs, buildToolsPath)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pkgPath := filepath.Join(buildToolsPath, entry.Name())
		versionEntries, err := afero.ReadDir(r.fs, pkgPath)
		if err != nil {
			continue
		}
		for _, vEntry := range versionEntries {
			if !vEntry.IsDir() {
				continue
			}
			binPath := filepath.Join(pkgPath, vEntry.Name(), "bin")
			if exists, _ := afero.DirExists(r.fs, binPath); exists {
				binPaths = append(binPaths, binPath)
			}
		}
	}

	return strings.Join(binPaths, ":")
}

func (r *ForgeRepository) chmodBuildScripts(sourcePath string) {
	exec.Command("chmod", "-R", "+x", filepath.Join(sourcePath, "build")).Run()
	exec.Command("chmod", "-R", "+x", filepath.Join(sourcePath, "ext")).Run()
}

func (r *ForgeRepository) buildConfigureMake(sourcePath, prefix string, config domain.ForgeConfig, env []string) (domain.Forge, error) {
	configurePath := filepath.Join(sourcePath, "configure")
	if _, err := os.Stat(configurePath); os.IsNotExist(err) {
		return domain.Forge{}, fmt.Errorf("configure script not found at %s", configurePath)
	}

	if err := os.Chmod(configurePath, 0o755); err != nil {
		return domain.Forge{}, fmt.Errorf("failed to chmod configure: %w", err)
	}

	if config.Name == "m4" {
		autoreconf := exec.Command("autoreconf", "-fi")
		autoreconf.Dir = sourcePath
		autoreconf.Env = env
		autoreconf.Stdout = os.Stdout
		autoreconf.Stderr = os.Stderr
		fmt.Println("Running autoreconf for m4")
		if err := autoreconf.Run(); err != nil {
			return domain.Forge{}, fmt.Errorf("autoreconf failed: %w", err)
		}
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
		return domain.Forge{}, fmt.Errorf("configure failed: %w", err)
	}

	if err := r.makeWithName(sourcePath, config.Jobs, env, config.Name); err != nil {
		return domain.Forge{}, err
	}

	if err := r.makeInstall(sourcePath, config.Jobs, env); err != nil {
		return domain.Forge{}, err
	}

	return domain.Forge{Prefix: prefix}, nil
}

func (r *ForgeRepository) buildMakeOnly(sourcePath, prefix string, config domain.ForgeConfig, env []string) (domain.Forge, error) {
	jobs := config.Jobs
	if jobs == 0 {
		jobs = runtime.NumCPU()
	}

	mk := exec.Command("make", fmt.Sprintf("-j%d", jobs))
	mk.Dir = sourcePath
	mk.Env = env
	mk.Stdout = os.Stdout
	mk.Stderr = os.Stderr

	fmt.Println("Running make for", config.Name)
	if err := mk.Run(); err != nil {
		return domain.Forge{}, fmt.Errorf("make failed: %w", err)
	}

	mkInstall := exec.Command("make", "install")
	mkInstall.Dir = sourcePath
	mkInstall.Env = env
	mkInstall.Stdout = os.Stdout
	mkInstall.Stderr = os.Stderr

	fmt.Println("Running make install for", config.Name)
	if err := mkInstall.Run(); err != nil {
		return domain.Forge{}, fmt.Errorf("make install failed: %w", err)
	}

	return domain.Forge{Prefix: prefix}, nil
}

func (r *ForgeRepository) buildCMake(sourcePath, prefix string, config domain.ForgeConfig, env []string) (domain.Forge, error) {
	buildDir := filepath.Join(sourcePath, "build")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return domain.Forge{}, fmt.Errorf("failed to create build directory: %w", err)
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
		return domain.Forge{}, fmt.Errorf("cmake failed: %w", err)
	}

	mk := exec.Command("make", fmt.Sprintf("-j%d", jobs))
	mk.Dir = buildDir
	mk.Env = env
	mk.Stdout = os.Stdout
	mk.Stderr = os.Stderr

	fmt.Println("Running make for", config.Name)
	if err := mk.Run(); err != nil {
		return domain.Forge{}, fmt.Errorf("make failed: %w", err)
	}

	mkInstall := exec.Command("make", "install")
	mkInstall.Dir = buildDir
	mkInstall.Env = env
	mkInstall.Stdout = os.Stdout
	mkInstall.Stderr = os.Stderr

	fmt.Println("Running make install for", config.Name)
	if err := mkInstall.Run(); err != nil {
		return domain.Forge{}, fmt.Errorf("make install failed: %w", err)
	}

	return domain.Forge{Prefix: prefix}, nil
}

func (r *ForgeRepository) buildAutogen(sourcePath, prefix string, config domain.ForgeConfig, env []string) (domain.Forge, error) {
	jobs := config.Jobs
	if jobs == 0 {
		jobs = runtime.NumCPU()
	}

	autogenPath := filepath.Join(sourcePath, "autogen.sh")
	if _, err := os.Stat(autogenPath); err == nil {
		autogen := exec.Command("./autogen.sh")
		autogen.Dir = sourcePath
		autogen.Env = env
		autogen.Stdout = os.Stdout
		autogen.Stderr = os.Stderr
		fmt.Println("Running autogen.sh for", config.Name)
		if err := autogen.Run(); err != nil {
			return domain.Forge{}, fmt.Errorf("autogen failed: %w", err)
		}
	}

	configurePath := filepath.Join(sourcePath, "configure")
	if _, err := os.Stat(configurePath); err == nil {
		if err := os.Chmod(configurePath, 0o755); err != nil {
			return domain.Forge{}, fmt.Errorf("failed to chmod configure: %w", err)
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
			return domain.Forge{}, fmt.Errorf("configure failed: %w", err)
		}
	}

	if err := r.makeWithName(sourcePath, jobs, env, config.Name); err != nil {
		return domain.Forge{}, err
	}

	if err := r.makeInstall(sourcePath, jobs, env); err != nil {
		return domain.Forge{}, err
	}

	return domain.Forge{Prefix: prefix}, nil
}

func (r *ForgeRepository) make(sourcePath string, jobs int, env []string) error {
	return r.makeWithName(sourcePath, jobs, env, "")
}

func (r *ForgeRepository) makeWithName(sourcePath string, jobs int, env []string, pkgName string) error {
	if jobs == 0 {
		jobs = runtime.NumCPU()
	}

	if pkgName == "m4" {
		env = append(env, "M4_MAINTAINER_MODE=no")
	}

	mk := exec.Command("make", fmt.Sprintf("-j%d", jobs))
	mk.Dir = sourcePath
	mk.Env = env
	mk.Stdout = os.Stdout
	mk.Stderr = os.Stderr

	fmt.Println("Running make for", sourcePath)
	if err := mk.Run(); err != nil {
		return fmt.Errorf("make failed: %w", err)
	}

	return nil
}

func (r *ForgeRepository) makeInstall(sourcePath string, jobs int, env []string) error {
	if jobs == 0 {
		jobs = runtime.NumCPU()
	}

	mkInstall := exec.Command("make", "install")
	mkInstall.Dir = sourcePath
	mkInstall.Env = env
	mkInstall.Stdout = os.Stdout
	mkInstall.Stderr = os.Stderr

	fmt.Println("Running make install for", sourcePath)
	if err := mkInstall.Run(); err != nil {
		return fmt.Errorf("make install failed: %w", err)
	}

	return nil
}
