package cmd

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/bsisduck/octo/internal/docker"
	"github.com/bsisduck/octo/internal/tui/analyze"
	"github.com/bsisduck/octo/internal/ui/format"
)

// AnalyzeOutput holds structured analyze data for JSON/YAML output
type AnalyzeOutput struct {
	Containers []docker.ContainerInfo `json:"containers" yaml:"containers"`
	Images     []docker.ImageInfo     `json:"images" yaml:"images"`
	Volumes    []docker.VolumeInfo    `json:"volumes" yaml:"volumes"`
	Networks   []docker.NetworkInfo   `json:"networks" yaml:"networks"`
	DiskUsage  *docker.DiskUsageInfo  `json:"diskUsage" yaml:"diskUsage"`
}

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze Docker resource usage",
	Long: `Analyze Docker resources with an interactive tree view:
- Explore containers, images, volumes, and networks
- View size breakdown and usage patterns
- Identify large or unused resources
- Navigate with arrow keys, delete with 'd'`,
	RunE: runAnalyze,
}

func init() {
	analyzeCmd.Flags().StringP("type", "t", "", "Filter by type: containers, images, volumes, networks")
	analyzeCmd.Flags().BoolP("dangling", "d", false, "Show only dangling/unused resources")
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	typeFilter, _ := cmd.Flags().GetString("type")
	dangling, _ := cmd.Flags().GetBool("dangling")
	outputFormat, _ := cmd.Flags().GetString("output-format")

	client, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("connecting to Docker: %w", err)
	}
	defer func() { _ = client.Close() }()

	// Non-TUI path for JSON/YAML output
	if outputFormat == "json" || outputFormat == "yaml" {
		return runAnalyzeCLI(client, outputFormat, typeFilter, dangling)
	}

	// TUI path (existing behavior)
	model := analyze.New(client, analyze.Options{
		TypeFilter: typeFilter,
		Dangling:   dangling,
	})

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("running analyze: %w", err)
	}
	return nil
}

func runAnalyzeCLI(client docker.DockerService, outputFormat, typeFilter string, dangling bool) error {
	ctx := context.Background()
	output := AnalyzeOutput{}

	// Fetch data based on type filter (empty = all types)
	if typeFilter == "" || typeFilter == "containers" || typeFilter == "container" || typeFilter == "c" {
		containers, err := client.ListContainers(ctx, true)
		if err != nil {
			return fmt.Errorf("listing containers: %w", err)
		}
		if dangling {
			stopped, _ := client.GetStoppedContainers(ctx)
			containers = stopped
		}
		output.Containers = containers
	}

	if typeFilter == "" || typeFilter == "images" || typeFilter == "image" || typeFilter == "i" {
		images, err := client.ListImages(ctx, true)
		if err != nil {
			return fmt.Errorf("listing images: %w", err)
		}
		if dangling {
			danglingImages, _ := client.GetDanglingImages(ctx)
			images = danglingImages
		}
		output.Images = images
	}

	if typeFilter == "" || typeFilter == "volumes" || typeFilter == "volume" || typeFilter == "v" {
		volumes, err := client.ListVolumes(ctx)
		if err != nil {
			return fmt.Errorf("listing volumes: %w", err)
		}
		if dangling {
			unused, _ := client.GetUnusedVolumes(ctx)
			volumes = unused
		}
		output.Volumes = volumes
	}

	if typeFilter == "" || typeFilter == "networks" || typeFilter == "network" || typeFilter == "n" {
		networks, err := client.ListNetworks(ctx)
		if err != nil {
			return fmt.Errorf("listing networks: %w", err)
		}
		output.Networks = networks
	}

	if typeFilter == "" {
		diskUsage, err := client.GetDiskUsage(ctx)
		if err != nil {
			return fmt.Errorf("getting disk usage: %w", err)
		}
		output.DiskUsage = diskUsage
	}

	switch outputFormat {
	case "json":
		return format.FormatJSON(os.Stdout, output)
	case "yaml":
		return format.FormatYAML(os.Stdout, output)
	}
	return nil
}
