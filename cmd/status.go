package cmd

import (
	"context"
	"fmt"

	"github.com/bsisduck/octo/internal/docker"
	"github.com/bsisduck/octo/internal/tui/status"
	"github.com/dustin/go-humanize"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Docker system status and resource usage",
	Long: `Display real-time Docker system status including:
- Running and stopped containers
- Image count and total size
- Volume usage
- Network status
- System resource consumption`,
	RunE: runStatus,
}

func init() {
	statusCmd.Flags().BoolP("watch", "w", false, "Continuously update status")
}

func runStatus(cmd *cobra.Command, args []string) error {
	watch, _ := cmd.Flags().GetBool("watch")

	client, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("connecting to Docker: %w", err)
	}
	defer client.Close()

	if watch {
		model := status.New(client, true)
		p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("running status: %w", err)
		}
		return nil
	}

	// One-shot status display
	ctx := context.Background()
	info, err := client.GetServerInfo(ctx)
	if err != nil {
		return fmt.Errorf("getting Docker info: %w", err)
	}
	diskUsage, err := client.GetDiskUsage(ctx)
	if err != nil {
		return fmt.Errorf("getting disk usage: %w", err)
	}

	// Print status (inline formatting for one-shot mode)
	fmt.Println()
	fmt.Printf("Docker System Status\n")
	fmt.Println("────────────────────────────────────────")
	fmt.Printf("Server Version: %s\n", info.ServerVersion)
	fmt.Printf("OS/Arch: %s (%s)\n", info.OperatingSystem, info.Architecture)
	fmt.Println()
	fmt.Printf("Containers\n")
	fmt.Printf("  Running: %d\n", info.ContainersRunning)
	fmt.Printf("  Paused: %d\n", info.ContainersPaused)
	fmt.Printf("  Stopped: %d\n", info.ContainersStopped)
	fmt.Printf("  Total: %d\n", info.Containers)
	fmt.Println()
	fmt.Printf("Images\n")
	fmt.Printf("  Total: %d\n", info.Images)
	fmt.Printf("  Size: %s\n", humanize.Bytes(uint64(diskUsage.Images)))
	fmt.Println()
	fmt.Printf("Volumes\n")
	volumes, _ := client.ListVolumes(ctx)
	unusedVolumes, _ := client.GetUnusedVolumes(ctx)
	fmt.Printf("  Total: %d\n", len(volumes))
	fmt.Printf("  Unused: %d\n", len(unusedVolumes))
	fmt.Printf("  Size: %s\n", humanize.Bytes(uint64(diskUsage.Volumes)))
	fmt.Println()
	fmt.Printf("Disk Usage\n")
	fmt.Printf("  Total: %s\n", humanize.Bytes(uint64(diskUsage.Total)))
	fmt.Printf("  Reclaimable: %s\n", humanize.Bytes(uint64(diskUsage.TotalReclaimable)))
	fmt.Printf("  Build Cache: %s\n", humanize.Bytes(uint64(diskUsage.BuildCache)))
	fmt.Println()
	return nil
}
