package terminal

import "fmt"

func NewNothingHandler() {
	fmt.Println("PHPV - PHP Version Manager")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("    phpv <command> [arguments]")
	fmt.Println()
	fmt.Println("COMMANDS:")
	fmt.Println("    list                        List installed PHP versions")
	fmt.Println("    download                    Download PHP source code")
	fmt.Println("    build                       Build and install PHP from source (using Clang)")
	fmt.Println("    prune                       Remove all build artifacts binaries, directories and files")
	fmt.Println("    help                        Show this help message")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("    phpv list                   # List installed versions")
	fmt.Println("    phpv list 8				 # List installed versions for major version 8")
	fmt.Println("    phpv list 8.3               # List installed versions for major.minor version 8.3")
	fmt.Println("    phpv download 8             # Download latest PHP 8.x source")
	fmt.Println("    phpv download 8.3           # Download latest PHP 8.3.x source")
	fmt.Println("    phpv download 8.4.14        # Download specific PHP 8.4.14 source")
	fmt.Println("    phpv build 8                # Build latest PHP 8.x from source")
	fmt.Println("    phpv build 8.3              # Build latest PHP 8.3.x from source")
	fmt.Println("    phpv build 8.4.14           # Build specific PHP 8.4.14 from source")
	fmt.Println("    phpv which                  # Show current PHP binary path")
	fmt.Println()
	fmt.Println("ENVIRONMENT VARIABLES:")
	fmt.Println("    PHPV_ROOT    Root directory for phpv (default: ~/.phpv)")
	fmt.Println("    PHP_SOURCE   Source for downloads: 'github' or 'official' (default: github)")
}
