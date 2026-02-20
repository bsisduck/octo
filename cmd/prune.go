package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"

	"github.com/bsisduck/octo/internal/docker"
	"github.com/bsisduck/octo/internal/ui/format"
	"github.com/bsisduck/octo/internal/ui/styles"
)

// PruneOutput holds structured prune data for JSON/YAML output
type PruneOutput struct {
	DryRun         bool          `json:"dry_run" yaml:"dry_run"`
	DiskBefore     PruneDisk     `json:"disk_before" yaml:"disk_before"`
	Results        []PruneResult `json:"results" yaml:"results"`
	TotalReclaimed uint64        `json:"total_reclaimed_bytes" yaml:"total_reclaimed_bytes"`
}

type PruneDisk struct {
	Images      int64 `json:"images_bytes" yaml:"images_bytes"`
	Containers  int64 `json:"containers_bytes" yaml:"containers_bytes"`
	Volumes     int64 `json:"volumes_bytes" yaml:"volumes_bytes"`
	BuildCache  int64 `json:"build_cache_bytes" yaml:"build_cache_bytes"`
	Total       int64 `json:"total_bytes" yaml:"total_bytes"`
	Reclaimable int64 `json:"reclaimable_bytes" yaml:"reclaimable_bytes"`
}

type PruneResult struct {
	Resource  string `json:"resource" yaml:"resource"`
	Reclaimed uint64 `json:"reclaimed_bytes" yaml:"reclaimed_bytes"`
	Error     string `json:"error,omitempty" yaml:"error,omitempty"`
}

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Deep cleanup - remove all unused Docker resources",
	Long: `Perform a deep cleanup of all unused Docker resources.
This is equivalent to 'docker system prune -a --volumes'.

WARNING: This will remove:
- All stopped containers
- All networks not used by at least one container
- All dangling images
- All dangling build cache
- All anonymous volumes not used by at least one container

Use --dry-run to preview what would be removed.`,
	RunE: runPrune,
}

func init() {
	pruneCmd.Flags().BoolP("force", "f", false, "Don't prompt for confirmation")
	pruneCmd.Flags().Bool("volumes", false, "Also prune anonymous volumes")
	pruneCmd.Flags().BoolP("all", "a", false, "Remove all unused images, not just dangling")
}

func runPrune(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")
	pruneVolumes, _ := cmd.Flags().GetBool("volumes")
	all, _ := cmd.Flags().GetBool("all")
	outputFormat, _ := cmd.Flags().GetString("output-format")

	client, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("error connecting to Docker: %w", err)
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	// Get disk usage before
	usageBefore, err := client.GetDiskUsage(ctx)
	if err != nil {
		return fmt.Errorf("error getting disk usage: %w", err)
	}

	// JSON/YAML output path
	if outputFormat == "json" || outputFormat == "yaml" {
		return runPruneStructured(cmd, client, ctx, usageBefore, outputFormat, pruneVolumes, all, force)
	}

	// Text output (default)
	titleStyle := styles.Title
	warnStyle := styles.Warn
	successStyle := styles.Success
	infoStyle := styles.Info

	fmt.Println()
	fmt.Println(titleStyle.Render("🐙 Octo Deep Prune"))
	fmt.Println(strings.Repeat("─", 50))

	if IsDryRun() {
		fmt.Println(styles.Warning.Render("DRY RUN MODE - No changes will be made"))
		fmt.Println()
	}

	fmt.Println()
	fmt.Println("Current Docker disk usage:")
	fmt.Printf("  Images:      %s\n", humanize.Bytes(uint64(usageBefore.Images)))
	fmt.Printf("  Containers:  %s\n", humanize.Bytes(uint64(usageBefore.Containers)))
	fmt.Printf("  Volumes:     %s\n", humanize.Bytes(uint64(usageBefore.Volumes)))
	fmt.Printf("  Build Cache: %s\n", humanize.Bytes(uint64(usageBefore.BuildCache)))
	fmt.Printf("  Total:       %s\n", humanize.Bytes(uint64(usageBefore.Total)))
	fmt.Printf("  Reclaimable: %s\n", successStyle.Render(humanize.Bytes(uint64(usageBefore.TotalReclaimable))))
	fmt.Println()

	if usageBefore.TotalReclaimable == 0 {
		fmt.Println(infoStyle.Render("No unused resources to clean up."))
		fmt.Println()
		return nil
	}

	// Warning message
	fmt.Println(warnStyle.Render("WARNING: This will remove:"))
	fmt.Println("  • All stopped containers")
	fmt.Println("  • All unused networks")
	if all {
		fmt.Println("  • All unused images (not just dangling)")
	} else {
		fmt.Println("  • All dangling images")
	}
	fmt.Println("  • All build cache")
	if pruneVolumes {
		fmt.Println("  • All anonymous volumes")
	}
	fmt.Println()

	if IsDryRun() {
		fmt.Println(infoStyle.Render(fmt.Sprintf("Would reclaim approximately: %s",
			humanize.Bytes(uint64(usageBefore.TotalReclaimable)))))
		fmt.Println()
		return nil
	}

	// Confirmation
	if !force {
		fmt.Printf("Are you sure you want to continue? [y/N]: ")
		var response string
		_, _ = fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println()
			fmt.Println(infoStyle.Render("Operation canceled."))
			fmt.Println()
			return nil
		}
	}

	fmt.Println()
	fmt.Println("Pruning Docker resources...")
	fmt.Println()

	var totalReclaimed uint64

	// Prune containers
	fmt.Print("  Pruning containers... ")
	containerReclaimed, err := client.PruneContainers(ctx)
	if err != nil {
		fmt.Println(warnStyle.Render(fmt.Sprintf("error: %v", err)))
	} else {
		totalReclaimed += containerReclaimed
		fmt.Println(successStyle.Render(fmt.Sprintf("done (%s)", humanize.Bytes(containerReclaimed))))
	}

	// Prune images
	fmt.Print("  Pruning images... ")
	imageReclaimed, err := client.PruneImages(ctx, all)
	if err != nil {
		fmt.Println(warnStyle.Render(fmt.Sprintf("error: %v", err)))
	} else {
		totalReclaimed += imageReclaimed
		fmt.Println(successStyle.Render(fmt.Sprintf("done (%s)", humanize.Bytes(imageReclaimed))))
	}

	// Prune volumes
	if pruneVolumes {
		fmt.Print("  Pruning volumes... ")
		var volumeReclaimed uint64
		volumeReclaimed, err = client.PruneVolumes(ctx)
		if err != nil {
			fmt.Println(warnStyle.Render(fmt.Sprintf("error: %v", err)))
		} else {
			totalReclaimed += volumeReclaimed
			fmt.Println(successStyle.Render(fmt.Sprintf("done (%s)", humanize.Bytes(volumeReclaimed))))
		}
	}

	// Prune networks
	fmt.Print("  Pruning networks... ")
	err = client.PruneNetworks(ctx)
	if err != nil {
		fmt.Println(warnStyle.Render(fmt.Sprintf("error: %v", err)))
	} else {
		fmt.Println(successStyle.Render("done"))
	}

	// Prune build cache
	fmt.Print("  Pruning build cache... ")
	cacheReclaimed, err := client.PruneBuildCache(ctx, all)
	if err != nil {
		fmt.Println(warnStyle.Render(fmt.Sprintf("error: %v", err)))
	} else {
		totalReclaimed += cacheReclaimed
		fmt.Println(successStyle.Render(fmt.Sprintf("done (%s)", humanize.Bytes(cacheReclaimed))))
	}

	fmt.Println()
	fmt.Println(strings.Repeat("─", 50))
	fmt.Println(successStyle.Render(fmt.Sprintf("Total space reclaimed: %s", humanize.Bytes(totalReclaimed))))
	fmt.Println()
	return nil
}

func runPruneStructured(_ *cobra.Command, client *docker.Client, ctx context.Context, usageBefore *docker.DiskUsageInfo, outputFormat string, pruneVolumes bool, all bool, force bool) error {
	diskBefore := PruneDisk{
		Images:      usageBefore.Images,
		Containers:  usageBefore.Containers,
		Volumes:     usageBefore.Volumes,
		BuildCache:  usageBefore.BuildCache,
		Total:       usageBefore.Total,
		Reclaimable: usageBefore.TotalReclaimable,
	}

	// Dry-run: output disk-before with empty results
	if IsDryRun() {
		output := PruneOutput{
			DryRun:     true,
			DiskBefore: diskBefore,
			Results:    []PruneResult{},
		}
		return formatPruneOutput(outputFormat, output)
	}

	// Confirmation for non-force mode: skip in JSON/YAML (require --force)
	if !force {
		return fmt.Errorf("--force flag is required for JSON/YAML output mode")
	}

	var results []PruneResult
	var totalReclaimed uint64

	// Prune containers
	containerReclaimed, err := client.PruneContainers(ctx)
	result := PruneResult{Resource: "containers", Reclaimed: containerReclaimed}
	if err != nil {
		result.Error = err.Error()
	} else {
		totalReclaimed += containerReclaimed
	}
	results = append(results, result)

	// Prune images
	imageReclaimed, err := client.PruneImages(ctx, all)
	result = PruneResult{Resource: "images", Reclaimed: imageReclaimed}
	if err != nil {
		result.Error = err.Error()
	} else {
		totalReclaimed += imageReclaimed
	}
	results = append(results, result)

	// Prune volumes
	if pruneVolumes {
		volumeReclaimed, volErr := client.PruneVolumes(ctx)
		result = PruneResult{Resource: "volumes", Reclaimed: volumeReclaimed}
		if volErr != nil {
			result.Error = volErr.Error()
		} else {
			totalReclaimed += volumeReclaimed
		}
		results = append(results, result)
	}

	// Prune networks
	netErr := client.PruneNetworks(ctx)
	result = PruneResult{Resource: "networks", Reclaimed: 0}
	if netErr != nil {
		result.Error = netErr.Error()
	}
	results = append(results, result)

	// Prune build cache
	cacheReclaimed, err := client.PruneBuildCache(ctx, all)
	result = PruneResult{Resource: "build_cache", Reclaimed: cacheReclaimed}
	if err != nil {
		result.Error = err.Error()
	} else {
		totalReclaimed += cacheReclaimed
	}
	results = append(results, result)

	output := PruneOutput{
		DryRun:         false,
		DiskBefore:     diskBefore,
		Results:        results,
		TotalReclaimed: totalReclaimed,
	}
	return formatPruneOutput(outputFormat, output)
}

func formatPruneOutput(outputFormat string, output PruneOutput) error {
	switch outputFormat {
	case "json":
		return format.FormatJSON(os.Stdout, output)
	case "yaml":
		return format.FormatYAML(os.Stdout, output)
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}
}
