// packrat is a CLI tool for backing up and restoring WSL home directories.
//
// Usage:
//
//	packrat backup [archive-path] [--dry-run] [--non-interactive]
//	packrat restore <archive-path> [destination] [--non-interactive]
//	packrat inspect <archive-path>
package main

import (
	"fmt"
	"os"

	"packrat/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
