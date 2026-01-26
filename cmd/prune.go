package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
)

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
	Run: runPrune,
}

func init() {
	pruneCmd.Flags().BoolP("force", "f", false, "Don't prompt for confirmation")
	pruneCmd.Flags().Bool("volumes", false, "Also prune anonymous volumes")
	pruneCmd.Flags().BoolP("all", "a", false, "Remove all unused images, not just dangling")
}

func runPrune(cmd *cobra.Command, args []string) {
	force, _ := cmd.Flags().GetBool("force")
	pruneVolumes, _ := cmd.Flags().GetBool("volumes")
	all, _ := cmd.Flags().GetBool("all")

	// Styles
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	warnStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	fmt.Println()
	fmt.Println(titleStyle.Render("🐙 Octo Deep Prune"))
	fmt.Println(strings.Repeat("─", 50))

	if IsDryRun() {
		fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("DRY RUN MODE - No changes will be made"))
		fmt.Println()
	}

	client, err := NewDockerClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to Docker: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Get disk usage before
	usageBefore, err := client.GetDiskUsage()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting disk usage: %v\n", err)
		os.Exit(1)
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
		return
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
		return
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
			return
		}
	}

	fmt.Println()
	fmt.Println("Pruning Docker resources...")
	fmt.Println()

	var totalReclaimed uint64

	// Prune containers
	fmt.Print("  Pruning containers... ")
	containerReclaimed, err := client.PruneContainers()
	if err != nil {
		fmt.Println(warnStyle.Render(fmt.Sprintf("error: %v", err)))
	} else {
		totalReclaimed += containerReclaimed
		fmt.Println(successStyle.Render(fmt.Sprintf("done (%s)", humanize.Bytes(containerReclaimed))))
	}

	// Prune images
	fmt.Print("  Pruning images... ")
	imageReclaimed, err := client.PruneImages(all)
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
		volumeReclaimed, err = client.PruneVolumes()
		if err != nil {
			fmt.Println(warnStyle.Render(fmt.Sprintf("error: %v", err)))
		} else {
			totalReclaimed += volumeReclaimed
			fmt.Println(successStyle.Render(fmt.Sprintf("done (%s)", humanize.Bytes(volumeReclaimed))))
		}
	}

	// Prune networks
	fmt.Print("  Pruning networks... ")
	err = client.PruneNetworks()
	if err != nil {
		fmt.Println(warnStyle.Render(fmt.Sprintf("error: %v", err)))
	} else {
		fmt.Println(successStyle.Render("done"))
	}

	// Prune build cache
	fmt.Print("  Pruning build cache... ")
	cacheReclaimed, err := client.PruneBuildCache(all)
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
}
