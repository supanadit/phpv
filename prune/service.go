package prune

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

// GetPhpvRoot returns the PHPV_ROOT directory
func (s *Service) GetPhpvRoot() string {
	root := viper.GetString("PHPV_ROOT")
	if root == "" {
		homeDir, _ := os.UserHomeDir()
		root = filepath.Join(homeDir, ".phpv")
	}
	return root
}

// Prune removes all build artifacts and cached files
func (s *Service) Prune() error {
	root := s.GetPhpvRoot()

	dirsToRemove := []string{
		filepath.Join(root, "dependencies"),
		filepath.Join(root, "dependencies-src"),
		filepath.Join(root, "sources"),
		filepath.Join(root, "versions"),
	}

	fmt.Println("This will remove the following directories:")
	for _, dir := range dirsToRemove {
		if _, err := os.Stat(dir); err == nil {
			fmt.Printf("  - %s\n", dir)
		}
	}
	fmt.Println()

	var removedCount int
	var errors []error

	for _, dir := range dirsToRemove {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		fmt.Printf("Removing %s...\n", dir)
		if err := os.RemoveAll(dir); err != nil {
			errors = append(errors, fmt.Errorf("failed to remove %s: %w", dir, err))
		} else {
			removedCount++
		}
	}

	if len(errors) > 0 {
		fmt.Println()
		for _, err := range errors {
			fmt.Printf("Error: %v\n", err)
		}
		return fmt.Errorf("failed to remove %d directories", len(errors))
	}

	fmt.Println()
	if removedCount > 0 {
		fmt.Printf("✓ Successfully removed %d directories\n", removedCount)
	} else {
		fmt.Println("Nothing to prune - directories don't exist")
	}

	return nil
}

// CleanSource removes only the source directory for a specific PHP version
// This is useful for testing - it removes the source but keeps dependencies
func (s *Service) CleanSource(version string) error {
	root := s.GetPhpvRoot()
	sourceDir := filepath.Join(root, "sources", version)

	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		fmt.Printf("Source directory doesn't exist: %s\n", sourceDir)
		fmt.Println("Nothing to clean")
		return nil
	}

	fmt.Printf("Removing source directory: %s\n", sourceDir)
	if err := os.RemoveAll(sourceDir); err != nil {
		return fmt.Errorf("failed to remove source directory: %w", err)
	}

	fmt.Printf("✓ Removed source for PHP %s (dependencies preserved)\n", version)
	fmt.Println("Run 'phpv download " + version + "' to re-extract from cache")
	return nil
}
