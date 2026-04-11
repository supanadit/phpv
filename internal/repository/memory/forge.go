package memory

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/viper"
	"github.com/supanadit/phpv/domain"
	"github.com/supanadit/phpv/download"
	"github.com/supanadit/phpv/internal/repository/http"
	"github.com/supanadit/phpv/source"
	"github.com/supanadit/phpv/unload"
)

type BuildRepository struct {
	downloadRepository download.DownloadRepository
	unloadRepository   unload.UnloadRepository
}

func NewForgeRepository(ur unload.UnloadRepository) *BuildRepository {
	return &BuildRepository{
		downloadRepository: http.NewDownloadRepository(),
		unloadRepository:   ur,
	}
}

func (r *BuildRepository) Build(config domain.ForgeConfig) (domain.Forge, error) {
	if config.Jobs == 0 {
		config.Jobs = runtime.NumCPU()
	}

	url, err := r.resolveURL(config)
	if err != nil {
		return domain.Forge{}, err
	}

	cachePath := r.getCachePath(config.Name, config.Version)
	if err := r.download(url, cachePath); err != nil {
		return domain.Forge{}, err
	}

	sourcePath := r.getSourcePath(config.Name, config.Version)
	_, err = r.extract(cachePath, sourcePath)
	if err != nil {
		return domain.Forge{}, err
	}

	prefix := config.Prefix
	if prefix == "" {
		prefix = r.getVersionsPath(config.Version)
	}

	r.chmodBuildScripts(sourcePath)

	env := r.buildEnv(config.Env)

	if err := r.configure(sourcePath, prefix, config.ConfigureFlags, env); err != nil {
		return domain.Forge{}, err
	}

	if err := r.make(sourcePath, config.Jobs, env); err != nil {
		return domain.Forge{}, err
	}

	if err := r.makeInstall(sourcePath, config.Jobs, env); err != nil {
		return domain.Forge{}, err
	}

	return domain.Forge{Prefix: prefix}, nil
}

func (r *BuildRepository) resolveURL(config domain.ForgeConfig) (string, error) {
	sourceRepository := NewSourceRepository()
	sourceService := source.NewService(sourceRepository)

	phps, err := sourceService.GetVersions()
	if err != nil {
		return "", err
	}

	for _, src := range phps {
		if src.Name == config.Name && src.Version == config.Version {
			return src.URL, nil
		}
	}

	return "", fmt.Errorf("source not found for %s version %s", config.Name, config.Version)
}

func (r *BuildRepository) getCachePath(name, version string) string {
	cacheDir := filepath.Join(viper.GetString("PHPV_ROOT"), "cache")
	return filepath.Join(cacheDir, fmt.Sprintf("%s-%s.tar.gz", name, version))
}

func (r *BuildRepository) getSourcePath(name, version string) string {
	sourceDir := filepath.Join(viper.GetString("PHPV_ROOT"), "sources")
	return filepath.Join(sourceDir, name, version)
}

func (r *BuildRepository) getVersionsPath(version string) string {
	versionsDir := filepath.Join(viper.GetString("PHPV_ROOT"), "versions")
	return filepath.Join(versionsDir, version)
}

func (r *BuildRepository) download(url, cachePath string) error {
	downloadHTTPSvc := download.NewService(r.downloadRepository)

	cacheDir := filepath.Dir(cachePath)
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}

	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		if _, err := downloadHTTPSvc.Download(url, cachePath); err != nil {
			return err
		}
		fmt.Println("Download completed:", cachePath)
	} else {
		fmt.Println("Using cached:", cachePath)
	}

	return nil
}

func (r *BuildRepository) extract(cachePath, sourcePath string) (*domain.Unload, error) {
	unloadSvc := unload.NewService(r.unloadRepository)

	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		sourceDir := filepath.Dir(sourcePath)
		if err := os.MkdirAll(sourceDir, 0o755); err != nil {
			return nil, err
		}

		result, err := unloadSvc.Unpack(cachePath, sourcePath)
		if err != nil {
			return nil, err
		}

		entries, _ := os.ReadDir(sourcePath)
		if len(entries) == 1 && entries[0].IsDir() {
			extractedFolder := filepath.Join(sourcePath, entries[0].Name())
			files, _ := os.ReadDir(extractedFolder)
			for _, f := range files {
				os.Rename(filepath.Join(extractedFolder, f.Name()), filepath.Join(sourcePath, f.Name()))
			}
			os.RemoveAll(extractedFolder)
		}

		fmt.Printf("Extracted %d files to: %s\n", result.Extracted, sourcePath)
	} else {
		fmt.Println("Using cached source:", sourcePath)
	}

	return nil, nil
}

func (r *BuildRepository) chmodBuildScripts(sourcePath string) {
	exec.Command("chmod", "-R", "+x", filepath.Join(sourcePath, "build")).Run()
	exec.Command("chmod", "-R", "+x", filepath.Join(sourcePath, "ext")).Run()
}

func (r *BuildRepository) buildEnv(env map[string]string) []string {
	if env == nil {
		return nil
	}
	e := make([]string, 0, len(env))
	for k, v := range env {
		e = append(e, fmt.Sprintf("%s=%s", k, v))
	}
	return e
}

func (r *BuildRepository) configure(sourcePath, prefix string, flags []string, env []string) error {
	if err := os.Chmod(filepath.Join(sourcePath, "configure"), 0o755); err != nil {
		return err
	}

	args := []string{fmt.Sprintf("--prefix=%s", prefix)}
	args = append(args, flags...)

	configure := exec.Command("./configure", args...)
	configure.Dir = sourcePath
	configure.Stdout = os.Stdout
	configure.Stderr = os.Stderr
	configure.Env = env

	fmt.Println("Starting configure...")
	if err := configure.Run(); err != nil {
		return fmt.Errorf("configure failed: %w", err)
	}

	return nil
}

func (r *BuildRepository) make(sourcePath string, jobs int, env []string) error {
	fmt.Println("Path Version", sourcePath)

	mk := exec.Command("/usr/bin/make", fmt.Sprintf("-j%d", jobs))
	mk.Dir = sourcePath
	mk.Stdout = os.Stdout
	mk.Stderr = os.Stderr
	mk.Env = env

	fmt.Println("Starting make...")
	if err := mk.Run(); err != nil {
		return fmt.Errorf("make failed: %w", err)
	}

	return nil
}

func (r *BuildRepository) makeInstall(sourcePath string, jobs int, env []string) error {
	mk := exec.Command("/usr/bin/make", fmt.Sprintf("-j%d", jobs), "install")
	mk.Dir = sourcePath
	mk.Stdout = os.Stdout
	mk.Stderr = os.Stderr
	mk.Env = env

	fmt.Println("Starting make install...")
	if err := mk.Run(); err != nil {
		return fmt.Errorf("make install failed: %w", err)
	}

	return nil
}
