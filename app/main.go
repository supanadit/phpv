package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/supanadit/phpv/repository"
	"github.com/supanadit/phpv/usecase"
)

func main() {
	// Command line flags
	simulate := flag.Bool("simulate", false, "Use simulated builds instead of real compilation")
	flag.Parse()

	ctx := context.Background()

	// Use ~/.phpv as the base directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get home directory: %v", err)
	}
	baseDir := filepath.Join(homeDir, ".phpv")

	// Initialize repositories
	versionRepo := repository.NewInMemoryPHPVersionRepository()
	installRepo := repository.NewInMemoryInstallationRepository()
	downloader := repository.NewHTTPDownloader()
	var builder usecase.Builder
	if *simulate {
		builder = repository.NewSimulatedSourceBuilder()
		fmt.Println("Using simulated builds (fast for testing)")
	} else {
		builder = repository.NewSourceBuilder()
		fmt.Println("Using real PHP compilation (this will take several minutes)")
	}
	filesystem := repository.NewOSFileSystem()

	// Initialize usecase
	installService := usecase.NewInstallationService(
		versionRepo,
		installRepo,
		downloader,
		builder,
		filesystem,
		baseDir,
	)

	// Demo: Install PHP 8.1.0
	// fmt.Println("Installing PHP 8.1.0...")
	// if err := installService.InstallVersion(ctx, "8.1.0"); err != nil {
	// 	log.Fatalf("Failed to install PHP 8.1.0: %v", err)
	// }
	// fmt.Println("✅ PHP 8.1.0 installed successfully")

	// Demo: Install PHP 5.6.0 (should show compatibility warning)
	fmt.Println("Installing PHP 5.6.0...")
	if err := installService.InstallVersion(ctx, "5.6.0"); err != nil {
		log.Fatalf("Failed to install PHP 5.6.0: %v", err)
	}
	fmt.Println("✅ PHP 5.6.0 installed successfully")

	// // Demo: Install PHP 7.0.0 (should show compatibility warning)
	// fmt.Println("Installing PHP 7.0.0...")
	// if err := installService.InstallVersion(ctx, "7.0.0"); err != nil {
	// 	log.Fatalf("Failed to install PHP 7.0.0: %v", err)
	// }
	// fmt.Println("✅ PHP 7.0.0 installed successfully")

	// // Demo: Install PHP 8.2.0
	// fmt.Println("Installing PHP 8.2.0...")
	// if err := installService.InstallVersion(ctx, "8.2.0"); err != nil {
	// 	log.Fatalf("Failed to install PHP 8.2.0: %v", err)
	// }
	// fmt.Println("✅ PHP 8.2.0 installed successfully")

	// // Demo: List installed versions
	// fmt.Println("Listing installed versions...")
	// installations, err := installService.ListInstalledVersions(ctx)
	// if err != nil {
	// 	log.Fatalf("Failed to list installations: %v", err)
	// }
	// for _, inst := range installations {
	// 	fmt.Printf("  - %s (installed at: %s, active: %t)\n",
	// 		inst.Version.Version, inst.InstalledAt.Format("2006-01-02 15:04:05"), inst.IsActive)
	// }

	// // Demo: Switch to PHP 8.2.0
	// fmt.Println("Switching to PHP 8.2.0...")
	// if err := installService.SwitchVersion(ctx, "8.2.0"); err != nil {
	// 	log.Fatalf("Failed to switch to PHP 8.2.0: %v", err)
	// }
	// fmt.Println("✅ Switched to PHP 8.2.0")

	// Demo: Get active version
	fmt.Println("Getting active version...")
	active, err := installService.GetActiveVersion(ctx)
	if err != nil {
		log.Fatalf("Failed to get active version: %v", err)
	}
	fmt.Printf("Active version: %s\n", active.Version.Version)

	fmt.Printf("\n🎉 Demo completed successfully! PHP binaries installed to: %s\n", baseDir)
}
