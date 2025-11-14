package terminal

import "fmt"

func NewNothingHandler() {
	fmt.Println("PHP Version Manager (phpv)")
	fmt.Println("Use --list-versions to see available PHP versions.")
}
