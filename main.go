// Octo - Docker container orchestration and management CLI
// Like an octopus managing multiple containers with ease.
package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/bsisduck/octo/cmd"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			// Restore terminal to sane state:
			// Show cursor, exit alt-screen buffer, reset text attributes
			fmt.Fprint(os.Stderr, "\033[?25h")
			fmt.Fprint(os.Stderr, "\033[?1049l")
			fmt.Fprint(os.Stderr, "\033[0m")

			fmt.Fprintf(os.Stderr, "\nocto: fatal error: %v\n", r)
			if os.Getenv("OCTO_DEBUG") == "1" {
				debug.PrintStack()
			}
			os.Exit(1)
		}
	}()

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
