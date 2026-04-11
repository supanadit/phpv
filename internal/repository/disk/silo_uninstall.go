package disk

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

func (r *SiloRepository) GetInstalledBuildTools() ([]string, error) {
	buildToolsPath := filepath.Join(r.silo.Root, "build-tools")
	entries, err := afero.ReadDir(r.fs, buildToolsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read build-tools directory: %w", err)
	}

	var tools []string
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
			tools = append(tools, entry.Name()+"@"+vEntry.Name())
		}
	}

	return tools, nil
}

func (r *SiloRepository) RemoveUnusedBuildTools(dryRun bool) ([]string, []string, error) {
	refs, err := r.loadBuildToolsRefs()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load build-tools refs: %w", err)
	}

	trackedTools := make(map[string]bool)
	for key := range refs {
		trackedTools[key] = true
	}

	installedTools, err := r.GetInstalledBuildTools()
	if err != nil {
		return nil, nil, err
	}

	var removed []string
	var wouldRemove []string

	for _, tool := range installedTools {
		if !trackedTools[tool] {
			if dryRun {
				wouldRemove = append(wouldRemove, tool)
			} else {
				parts := strings.Split(tool, "@")
				if len(parts) == 2 {
					name, version := parts[0], parts[1]
					toolPath := filepath.Join(r.silo.Root, "build-tools", name, version)
					if err := r.fs.RemoveAll(toolPath); err != nil {
						return nil, nil, fmt.Errorf("failed to remove %s: %w", tool, err)
					}
					removed = append(removed, tool)
				}
			}
		}
	}

	return removed, wouldRemove, nil
}
