// Octo - Docker container orchestration and management CLI
// Like an octopus managing multiple containers with ease.
package main

import (
	"os"

	"github.com/bsisduck/octo/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
