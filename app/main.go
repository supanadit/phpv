package main

import (
	"context"
	"fmt"
	"log"

	"github.com/supanadit/phpv/repository"
	"github.com/supanadit/phpv/usecase"
)

func main() {
	ctx := context.Background()

	// Initialize repositories
	versionRepo := repository.NewInMemoryPHPVersionRepository()
	installRepo := repository.NewInMemoryInstallationRepository()
	downloader := repository.NewHTTPDownloader()
	builder := repository.NewSourceBuilder()
	filesystem := repository.NewOSFileSystem()

	// Initialize usecase
	installService := usecase.NewInstallationService(
		versionRepo,
		installRepo,
		downloader,
		builder,
		filesystem,
		"/tmp/phpv-demo",
	)

	// Demo: Install PHP 8.1.0
	fmt.Println("Installing PHP 8.1.0...")
	if err := installService.InstallVersion(ctx, "8.1.0"); err != nil {
		log.Fatalf("Failed to install PHP 8.1.0: %v", err)
	}
	fmt.Println("✅ PHP 8.1.0 installed successfully")

	// Demo: Install PHP 8.2.0
	fmt.Println("Installing PHP 8.2.0...")
	if err := installService.InstallVersion(ctx, "8.2.0"); err != nil {
		log.Fatalf("Failed to install PHP 8.2.0: %v", err)
	}
	fmt.Println("✅ PHP 8.2.0 installed successfully")

	// Demo: List installed versions
	fmt.Println("Listing installed versions...")
	installations, err := installService.ListInstalledVersions(ctx)
	if err != nil {
		log.Fatalf("Failed to list installations: %v", err)
	}
	for _, inst := range installations {
		fmt.Printf("  - %s (installed at: %s, active: %t)\n",
			inst.Version.Version, inst.InstalledAt.Format("2006-01-02 15:04:05"), inst.IsActive)
	}

	// Demo: Switch to PHP 8.2.0
	fmt.Println("Switching to PHP 8.2.0...")
	if err := installService.SwitchVersion(ctx, "8.2.0"); err != nil {
		log.Fatalf("Failed to switch to PHP 8.2.0: %v", err)
	}
	fmt.Println("✅ Switched to PHP 8.2.0")

	// Demo: Get active version
	fmt.Println("Getting active version...")
	active, err := installService.GetActiveVersion(ctx)
	if err != nil {
		log.Fatalf("Failed to get active version: %v", err)
	}
	fmt.Printf("Active version: %s\n", active.Version.Version)

	fmt.Println("\n🎉 Demo completed successfully!")
}
