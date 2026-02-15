package cmd

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"

	"github.com/bsisduck/octo/internal/docker"
	"github.com/bsisduck/octo/internal/tui/status"
	"github.com/bsisduck/octo/internal/ui/format"
)

// StatusOutput holds structured status data for JSON/YAML output
type StatusOutput struct {
	ServerVersion string          `json:"server_version" yaml:"server_version"`
	OS            string          `json:"os" yaml:"os"`
	Arch          string          `json:"arch" yaml:"arch"`
	Containers    ContainerStatus `json:"containers" yaml:"containers"`
	Images        ImageStatus     `json:"images" yaml:"images"`
	Volumes       VolumeStatus    `json:"volumes" yaml:"volumes"`
	DiskUsage     DiskStatus      `json:"disk_usage" yaml:"disk_usage"`
}

type ContainerStatus struct {
	Running int `json:"running" yaml:"running"`
	Paused  int `json:"paused" yaml:"paused"`
	Stopped int `json:"stopped" yaml:"stopped"`
	Total   int `json:"total" yaml:"total"`
}

type ImageStatus struct {
	Total int   `json:"total" yaml:"total"`
	Size  int64 `json:"size_bytes" yaml:"size_bytes"`
}

type VolumeStatus struct {
	Total  int   `json:"total" yaml:"total"`
	Unused int   `json:"unused" yaml:"unused"`
	Size   int64 `json:"size_bytes" yaml:"size_bytes"`
}

type DiskStatus struct {
	Total       int64 `json:"total_bytes" yaml:"total_bytes"`
	Reclaimable int64 `json:"reclaimable_bytes" yaml:"reclaimable_bytes"`
	BuildCache  int64 `json:"build_cache_bytes" yaml:"build_cache_bytes"`
}

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

	volumes, _ := client.ListVolumes(ctx)
	unusedVolumes, _ := client.GetUnusedVolumes(ctx)

	// Check output format
	outputFormat, _ := cmd.Flags().GetString("output-format")
	switch outputFormat {
	case "json":
		return format.FormatJSON(os.Stdout, StatusOutput{
			ServerVersion: info.ServerVersion,
			OS:            info.OperatingSystem,
			Arch:          info.Architecture,
			Containers: ContainerStatus{
				Running: info.ContainersRunning,
				Paused:  info.ContainersPaused,
				Stopped: info.ContainersStopped,
				Total:   info.Containers,
			},
			Images: ImageStatus{
				Total: info.Images,
				Size:  diskUsage.Images,
			},
			Volumes: VolumeStatus{
				Total:  len(volumes),
				Unused: len(unusedVolumes),
				Size:   diskUsage.Volumes,
			},
			DiskUsage: DiskStatus{
				Total:       diskUsage.Total,
				Reclaimable: diskUsage.TotalReclaimable,
				BuildCache:  diskUsage.BuildCache,
			},
		})
	case "yaml":
		return format.FormatYAML(os.Stdout, StatusOutput{
			ServerVersion: info.ServerVersion,
			OS:            info.OperatingSystem,
			Arch:          info.Architecture,
			Containers: ContainerStatus{
				Running: info.ContainersRunning,
				Paused:  info.ContainersPaused,
				Stopped: info.ContainersStopped,
				Total:   info.Containers,
			},
			Images: ImageStatus{
				Total: info.Images,
				Size:  diskUsage.Images,
			},
			Volumes: VolumeStatus{
				Total:  len(volumes),
				Unused: len(unusedVolumes),
				Size:   diskUsage.Volumes,
			},
			DiskUsage: DiskStatus{
				Total:       diskUsage.Total,
				Reclaimable: diskUsage.TotalReclaimable,
				BuildCache:  diskUsage.BuildCache,
			},
		})
	}

	// Text output (default)
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
