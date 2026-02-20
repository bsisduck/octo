package cmd

import (
	"context"
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/bsisduck/octo/internal/docker"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  "Display Octo version, build information, and Docker connection status.",
	RunE:  runVersion,
}

func runVersion(cmd *cobra.Command, args []string) error {
	fmt.Printf("Octo version %s\n", Version)
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)

	if BuildTime != "" {
		fmt.Printf("Build time: %s\n", BuildTime)
	}
	if GitCommit != "" {
		fmt.Printf("Git commit: %s\n", GitCommit)
	}

	// Check Docker connection
	client, err := docker.NewClient()
	if err != nil {
		fmt.Printf("Docker: Not connected (%v)\n", err)
		return nil
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()
	info, err := client.GetServerInfo(ctx)
	if err != nil {
		fmt.Printf("Docker: Error getting info (%v)\n", err)
		return nil
	}

	fmt.Printf("Docker version: %s\n", info.ServerVersion)
	fmt.Printf("Docker OS: %s\n", info.OperatingSystem)
	fmt.Printf("Docker Arch: %s\n", info.Architecture)
	fmt.Printf("Containers: %d (running: %d)\n", info.Containers, info.ContainersRunning)
	fmt.Printf("Images: %d\n", info.Images)
	return nil
}
