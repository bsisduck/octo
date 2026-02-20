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

// CleanupOutput holds structured cleanup data for JSON/YAML output
type CleanupOutput struct {
	DryRun         bool             `json:"dry_run" yaml:"dry_run"`
	Sections       []CleanupSection `json:"sections" yaml:"sections"`
	TotalReclaimed uint64           `json:"total_reclaimed_bytes" yaml:"total_reclaimed_bytes"`
}

// CleanupSection holds data for a single cleanup category
type CleanupSection struct {
	Name      string   `json:"name" yaml:"name"`
	Found     int      `json:"found" yaml:"found"`
	Removed   int      `json:"removed" yaml:"removed"`
	Reclaimed uint64   `json:"reclaimed_bytes" yaml:"reclaimed_bytes"`
	Items     []string `json:"items" yaml:"items"`
	Skipped   bool     `json:"skipped" yaml:"skipped"`
}

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Smart cleanup of Docker resources",
	Long: `Intelligently clean up Docker resources with safety checks:
- Remove stopped containers
- Remove dangling images
- Remove unused volumes
- Remove unused networks
- Clear build cache

Use --dry-run to preview what would be removed without making changes.`,
	RunE: runCleanup,
}

func init() {
	cleanupCmd.Flags().Bool("all", false, "Remove all unused resources (not just dangling)")
	cleanupCmd.Flags().Bool("containers", false, "Remove stopped containers only")
	cleanupCmd.Flags().Bool("images", false, "Remove dangling images only")
	cleanupCmd.Flags().Bool("volumes", false, "Remove unused volumes only")
	cleanupCmd.Flags().Bool("networks", false, "Remove unused networks only")
	cleanupCmd.Flags().Bool("build-cache", false, "Remove build cache only")
	cleanupCmd.Flags().BoolP("force", "f", false, "Don't prompt for confirmation")
}

func runCleanup(cmd *cobra.Command, args []string) error {
	all, _ := cmd.Flags().GetBool("all")
	containersOnly, _ := cmd.Flags().GetBool("containers")
	imagesOnly, _ := cmd.Flags().GetBool("images")
	volumesOnly, _ := cmd.Flags().GetBool("volumes")
	networksOnly, _ := cmd.Flags().GetBool("networks")
	buildCacheOnly, _ := cmd.Flags().GetBool("build-cache")
	force, _ := cmd.Flags().GetBool("force")
	outputFormat, _ := cmd.Flags().GetString("output-format")

	// If no specific flag, clean all
	cleanAll := !containersOnly && !imagesOnly && !volumesOnly && !networksOnly && !buildCacheOnly

	structured := outputFormat == "json" || outputFormat == "yaml"

	client, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("connecting to Docker: %w", err)
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	// Get current disk usage for comparison
	initialUsage, diskUsageErr := client.GetDiskUsage(ctx)

	var totalReclaimed uint64
	var output CleanupOutput
	if structured {
		output.DryRun = IsDryRun()
	}

	// Styles (defined in internal/ui/styles/theme.go) - only needed for text output
	titleStyle := styles.Title
	sectionStyle := styles.Section
	successStyle := styles.Success
	warnStyle := styles.Warning
	infoStyle := styles.Info

	if !structured {
		fmt.Println()
		fmt.Println(titleStyle.Render("🐙 Octo Cleanup"))
		fmt.Println(strings.Repeat("─", 50))

		if IsDryRun() {
			fmt.Println(warnStyle.Render("DRY RUN MODE - No changes will be made"))
			fmt.Println()
		}
	}

	if diskUsageErr != nil && !structured {
		fmt.Printf("  %s\n", warnStyle.Render(fmt.Sprintf("Warning: could not get initial disk usage: %v", diskUsageErr)))
		fmt.Println()
	}

	// Clean stopped containers
	if cleanAll || containersOnly {
		if !structured {
			fmt.Println()
			fmt.Println(sectionStyle.Render("Stopped Containers"))
		}

		stopped, err := client.GetStoppedContainers(ctx)
		if err != nil {
			if !structured {
				fmt.Printf("  %s\n", warnStyle.Render(fmt.Sprintf("Error: %v", err)))
			}
		} else if structured {
			section := CleanupSection{
				Name:  "stopped_containers",
				Found: len(stopped),
				Items: make([]string, 0, len(stopped)),
			}
			for _, c := range stopped {
				section.Items = append(section.Items, fmt.Sprintf("%s (%s)", c.Name, c.ID))
			}
			if !IsDryRun() && len(stopped) > 0 {
				reclaimed, pruneErr := client.PruneContainers(ctx)
				if pruneErr == nil {
					section.Removed = len(stopped)
					section.Reclaimed = reclaimed
					totalReclaimed += reclaimed
				}
			}
			output.Sections = append(output.Sections, section)
		} else if len(stopped) == 0 {
			fmt.Printf("  %s\n", successStyle.Render("✓ No stopped containers"))
		} else {
			fmt.Printf("  Found %d stopped containers\n", len(stopped))
			for _, c := range stopped {
				fmt.Printf("    • %s (%s) - %s\n", c.Name, c.ID, c.Status)
			}

			if !IsDryRun() && (force || confirmAction("Remove stopped containers?")) {
				reclaimed, err := client.PruneContainers(ctx)
				if err != nil {
					fmt.Printf("  %s\n", warnStyle.Render(fmt.Sprintf("Error: %v", err)))
				} else {
					totalReclaimed += reclaimed
					fmt.Printf("  %s\n", successStyle.Render(fmt.Sprintf("✓ Removed %d containers, reclaimed %s",
						len(stopped), humanize.Bytes(reclaimed))))
				}
			} else if IsDryRun() {
				fmt.Printf("  %s\n", infoStyle.Render(fmt.Sprintf("→ Would remove %d containers", len(stopped))))
			}
		}
	}

	// Clean dangling images
	if cleanAll || imagesOnly {
		if !structured {
			fmt.Println()
			fmt.Println(sectionStyle.Render("Dangling Images"))
		}

		dangling, err := client.GetDanglingImages(ctx)
		if err != nil {
			if !structured {
				fmt.Printf("  %s\n", warnStyle.Render(fmt.Sprintf("Error: %v", err)))
			}
		} else if structured {
			section := CleanupSection{
				Name:  "dangling_images",
				Found: len(dangling),
				Items: make([]string, 0, len(dangling)),
			}
			for _, img := range dangling {
				section.Items = append(section.Items, fmt.Sprintf("%s (%s)", img.ID, humanize.Bytes(uint64(img.Size))))
			}
			if !IsDryRun() && len(dangling) > 0 {
				reclaimed, pruneErr := client.PruneImages(ctx, all)
				if pruneErr == nil {
					section.Removed = len(dangling)
					section.Reclaimed = reclaimed
					totalReclaimed += reclaimed
				}
			}
			output.Sections = append(output.Sections, section)
		} else if len(dangling) == 0 {
			fmt.Printf("  %s\n", successStyle.Render("✓ No dangling images"))
		} else {
			var totalSize int64
			for _, img := range dangling {
				totalSize += img.Size
				fmt.Printf("    • %s (%s)\n", img.ID, humanize.Bytes(uint64(img.Size)))
			}

			if !IsDryRun() && (force || confirmAction("Remove dangling images?")) {
				reclaimed, err := client.PruneImages(ctx, all)
				if err != nil {
					fmt.Printf("  %s\n", warnStyle.Render(fmt.Sprintf("Error: %v", err)))
				} else {
					totalReclaimed += reclaimed
					fmt.Printf("  %s\n", successStyle.Render(fmt.Sprintf("✓ Removed %d images, reclaimed %s",
						len(dangling), humanize.Bytes(reclaimed))))
				}
			} else if IsDryRun() {
				fmt.Printf("  %s\n", infoStyle.Render(fmt.Sprintf("→ Would remove %d images (%s)",
					len(dangling), humanize.Bytes(uint64(totalSize)))))
			}
		}
	}

	// Clean unused volumes
	if cleanAll || volumesOnly {
		if !structured {
			fmt.Println()
			fmt.Println(sectionStyle.Render("Unused Volumes"))
		}

		unused, err := client.GetUnusedVolumes(ctx)
		if err != nil {
			if !structured {
				fmt.Printf("  %s\n", warnStyle.Render(fmt.Sprintf("Error: %v", err)))
			}
		} else if structured {
			section := CleanupSection{
				Name:  "unused_volumes",
				Found: len(unused),
				Items: make([]string, 0, len(unused)),
			}
			for _, v := range unused {
				section.Items = append(section.Items, v.Name)
			}
			if !IsDryRun() && len(unused) > 0 {
				reclaimed, pruneErr := client.PruneVolumes(ctx)
				if pruneErr == nil {
					section.Removed = len(unused)
					section.Reclaimed = reclaimed
					totalReclaimed += reclaimed
				}
			}
			output.Sections = append(output.Sections, section)
		} else if len(unused) == 0 {
			fmt.Printf("  %s\n", successStyle.Render("✓ No unused volumes"))
		} else {
			fmt.Printf("  Found %d unused volumes\n", len(unused))
			for _, v := range unused {
				sizeStr := ""
				if v.Size > 0 {
					sizeStr = fmt.Sprintf(" (%s)", humanize.Bytes(uint64(v.Size)))
				}
				fmt.Printf("    • %s%s\n", v.Name, sizeStr)
			}

			if !IsDryRun() && (force || confirmAction("Remove unused volumes?")) {
				reclaimed, err := client.PruneVolumes(ctx)
				if err != nil {
					fmt.Printf("  %s\n", warnStyle.Render(fmt.Sprintf("Error: %v", err)))
				} else {
					totalReclaimed += reclaimed
					fmt.Printf("  %s\n", successStyle.Render(fmt.Sprintf("✓ Removed %d volumes, reclaimed %s",
						len(unused), humanize.Bytes(reclaimed))))
				}
			} else if IsDryRun() {
				fmt.Printf("  %s\n", infoStyle.Render(fmt.Sprintf("→ Would remove %d volumes", len(unused))))
			}
		}
	}

	// Clean unused networks
	if cleanAll || networksOnly {
		if !structured {
			fmt.Println()
			fmt.Println(sectionStyle.Render("Unused Networks"))
		}

		networks, err := client.ListNetworks(ctx)
		if err != nil {
			if !structured {
				fmt.Printf("  %s\n", warnStyle.Render(fmt.Sprintf("Error: %v", err)))
			}
		} else {
			unused := []docker.NetworkInfo{}
			for _, n := range networks {
				// Skip default networks
				if n.Name == "bridge" || n.Name == "host" || n.Name == "none" {
					continue
				}
				if n.Containers == 0 {
					unused = append(unused, n)
				}
			}

			if structured {
				section := CleanupSection{
					Name:  "unused_networks",
					Found: len(unused),
					Items: make([]string, 0, len(unused)),
				}
				for _, n := range unused {
					section.Items = append(section.Items, fmt.Sprintf("%s (%s)", n.Name, n.ID))
				}
				if !IsDryRun() && len(unused) > 0 {
					pruneErr := client.PruneNetworks(ctx)
					if pruneErr == nil {
						section.Removed = len(unused)
					}
				}
				output.Sections = append(output.Sections, section)
			} else if len(unused) == 0 {
				fmt.Printf("  %s\n", successStyle.Render("✓ No unused networks"))
			} else {
				fmt.Printf("  Found %d unused networks\n", len(unused))
				for _, n := range unused {
					fmt.Printf("    • %s (%s)\n", n.Name, n.ID)
				}

				if !IsDryRun() && (force || confirmAction("Remove unused networks?")) {
					err := client.PruneNetworks(ctx)
					if err != nil {
						fmt.Printf("  %s\n", warnStyle.Render(fmt.Sprintf("Error: %v", err)))
					} else {
						fmt.Printf("  %s\n", successStyle.Render(fmt.Sprintf("✓ Removed %d networks", len(unused))))
					}
				} else if IsDryRun() {
					fmt.Printf("  %s\n", infoStyle.Render(fmt.Sprintf("→ Would remove %d networks", len(unused))))
				}
			}
		}
	}

	// Clean build cache
	if cleanAll || buildCacheOnly {
		if !structured {
			fmt.Println()
			fmt.Println(sectionStyle.Render("Build Cache"))
		}

		if initialUsage != nil && initialUsage.BuildCache > 0 {
			if structured {
				section := CleanupSection{
					Name:  "build_cache",
					Found: 1,
					Items: []string{humanize.Bytes(uint64(initialUsage.BuildCache))},
				}
				if !IsDryRun() {
					reclaimed, pruneErr := client.PruneBuildCache(ctx, all)
					if pruneErr == nil {
						section.Removed = 1
						section.Reclaimed = reclaimed
						totalReclaimed += reclaimed
					}
				}
				output.Sections = append(output.Sections, section)
			} else {
				fmt.Printf("  Current build cache: %s\n", humanize.Bytes(uint64(initialUsage.BuildCache)))

				if !IsDryRun() && (force || confirmAction("Clear build cache?")) {
					reclaimed, err := client.PruneBuildCache(ctx, all)
					if err != nil {
						fmt.Printf("  %s\n", warnStyle.Render(fmt.Sprintf("Error: %v", err)))
					} else {
						totalReclaimed += reclaimed
						fmt.Printf("  %s\n", successStyle.Render(fmt.Sprintf("✓ Cleared build cache, reclaimed %s",
							humanize.Bytes(reclaimed))))
					}
				} else if IsDryRun() {
					fmt.Printf("  %s\n", infoStyle.Render(fmt.Sprintf("→ Would clear %s of build cache",
						humanize.Bytes(uint64(initialUsage.BuildCache)))))
				}
			}
		} else {
			if structured {
				output.Sections = append(output.Sections, CleanupSection{
					Name:    "build_cache",
					Found:   0,
					Items:   []string{},
					Skipped: true,
				})
			} else {
				fmt.Printf("  %s\n", successStyle.Render("✓ No build cache"))
			}
		}
	}

	// Output structured format
	if structured {
		output.TotalReclaimed = totalReclaimed
		switch outputFormat {
		case "json":
			return format.FormatJSON(os.Stdout, output)
		case "yaml":
			return format.FormatYAML(os.Stdout, output)
		}
	}

	// Text summary
	fmt.Println()
	fmt.Println(strings.Repeat("─", 50))
	if IsDryRun() {
		fmt.Println(warnStyle.Render("DRY RUN - No changes were made"))
	} else if totalReclaimed > 0 {
		fmt.Println(successStyle.Render(fmt.Sprintf("Total space reclaimed: %s", humanize.Bytes(totalReclaimed))))
	} else {
		fmt.Println(infoStyle.Render("No space reclaimed"))
	}
	fmt.Println()
	return nil
}

func confirmAction(prompt string) bool {
	fmt.Printf("  %s [y/N]: ", prompt)
	var response string
	_, _ = fmt.Scanln(&response)
	return strings.ToLower(response) == "y" || strings.ToLower(response) == "yes"
}
