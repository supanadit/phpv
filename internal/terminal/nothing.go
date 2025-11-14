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
	fmt.Println("    help                        Show this help message")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("    phpv list                   # List installed versions")
	fmt.Println("    phpv list 8				 # List installed versions for major version 8")
	fmt.Println("    phpv list 8.3               # List installed versions for major.minor version 8.3")
	fmt.Println("    phpv which                  # Show current PHP binary path")
	fmt.Println()
	fmt.Println("ENVIRONMENT VARIABLES:")
	fmt.Println("    PHPV_ROOT    Root directory for phpv (default: ~/.phpv)")
}
