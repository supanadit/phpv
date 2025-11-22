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
