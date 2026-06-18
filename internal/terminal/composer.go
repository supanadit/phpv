package terminal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type ComposerConfig struct {
	Config struct {
		Platform struct {
			PHP string `json:"php"`
		} `json:"platform"`
		PHPExtensions []string `json:"php-extensions"`
	} `json:"config"`
}

var versionExtractRegex = regexp.MustCompile(`(\d+\.?\d*\.?\d*)`)

func parseComposerJSON(dir string) (string, error) {
	composerPath := filepath.Join(dir, "composer.json")

	data, err := os.ReadFile(composerPath)
	if err != nil {
		return "", err
	}

	var config ComposerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return "", err
	}

	phpVersion := config.Config.Platform.PHP
	if phpVersion == "" {
		return "", nil
	}

	matches := versionExtractRegex.FindStringSubmatch(phpVersion)
	if matches == nil || matches[1] == "" {
		return "", nil
	}

	cleaned := matches[1]

	parts := strings.Split(cleaned, ".")
	if len(parts) >= 2 {
		cleaned = parts[0] + "." + parts[1]
		if len(parts) >= 3 && parts[2] != "" {
			cleaned += "." + parts[2]
		}
	}

	return cleaned, nil
}

func findComposerJSONFromPath(path string) (string, string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", "", err
	}

	dir := absPath
	for {
		version, err := parseComposerJSON(dir)
		if err == nil && version != "" {
			return dir, version, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", "", nil
}

func parseComposerExtensions(dir string) ([]string, error) {
	composerPath := filepath.Join(dir, "composer.json")

	data, err := os.ReadFile(composerPath)
	if err != nil {
		return nil, err
	}

	var config ComposerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return config.Config.PHPExtensions, nil
}

func findComposerExtensionsFromPath(path string) (string, []string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", nil, err
	}

	dir := absPath
	for {
		extensions, err := parseComposerExtensions(dir)
		if err == nil && len(extensions) > 0 {
			return dir, extensions, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", nil, nil
}

func parsePhpvrc(dir string) (string, error) {
	phpvrcPath := filepath.Join(dir, ".phpvrc")

	data, err := os.ReadFile(phpvrcPath)
	if err != nil {
		return "", err
	}

	version := strings.TrimSpace(string(data))
	if version == "" {
		return "", nil
	}

	return version, nil
}

func findPhpvrcFromPath(path string) (string, string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", "", err
	}

	dir := absPath
	for {
		version, err := parsePhpvrc(dir)
		if err == nil && version != "" {
			return dir, version, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", "", nil
}
