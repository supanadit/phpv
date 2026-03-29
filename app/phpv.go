package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/supanadit/phpv/advisor"
	"github.com/supanadit/phpv/internal/repository/disk"
)

func main() {
	viper.AutomaticEnv()
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic("You must have a home directory set for phpv to work")
	}
	viper.SetDefault("PHPV_ROOT", filepath.Join(homeDir, ".phpv"))

	repo := disk.NewAdvisorRepository()
	svc := advisor.NewAdvisorService(repo)

	tests := []struct {
		name    string
		version string
	}{
		{"php", "8.3.0"},
		{"php", "8.0.0"},
		{"openssl", "3.3.2"},
		{"libxml2", "2.12.7"},
		{"zlib", "1.3.1"},
	}

	for _, tc := range tests {
		check, err := svc.Check(tc.name, tc.version)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		fmt.Printf("Check: %s\n", check)
		fmt.Printf("  State: %s\n", check.State)
		fmt.Printf("  Action: %s\n", check.Action)
		fmt.Printf("  SystemAvailable: %v (%s)\n", check.SystemAvailable, check.SystemPath)
		fmt.Printf("  Message: %s\n\n", check.Message)
	}
}
