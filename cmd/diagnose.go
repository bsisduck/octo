package cmd

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
)

var diagnoseCmd = &cobra.Command{
	Use:   "diagnose",
	Short: "Diagnose Docker daemon health and configuration",
	Long: `Run diagnostic checks on Docker daemon:
- Connection status
- Daemon configuration
- Resource usage
- Potential issues
- Performance recommendations`,
	Run: runDiagnose,
}

func init() {
	diagnoseCmd.Flags().Bool("verbose", false, "Show detailed diagnostic information")
}

type DiagnosticResult struct {
	Name    string
	Status  string // "ok", "warn", "error"
	Message string
	Details string
}

func runDiagnose(cmd *cobra.Command, args []string) {
	verbose, _ := cmd.Flags().GetBool("verbose")

	// Styles
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	fmt.Println()
	fmt.Println(titleStyle.Render("🐙 Octo Docker Diagnostics"))
	fmt.Println(strings.Repeat("─", 50))
	fmt.Println()

	results := []DiagnosticResult{}

	// Check 1: Docker connection
	fmt.Print("Checking Docker connection... ")
	client, err := NewDockerClient()
	if err != nil {
		fmt.Println(errorStyle.Render("FAILED"))
		results = append(results, DiagnosticResult{
			Name:    "Docker Connection",
			Status:  "error",
			Message: "Cannot connect to Docker daemon",
			Details: err.Error(),
		})
		printDiagnosticSummary(results, verbose)
		os.Exit(1)
	}
	fmt.Println(okStyle.Render("OK"))
	results = append(results, DiagnosticResult{
		Name:    "Docker Connection",
		Status:  "ok",
		Message: "Connected to Docker daemon",
	})
	defer client.Close()

	// Check 2: Docker version
	fmt.Print("Checking Docker version... ")
	info, err := client.GetServerInfo()
	if err != nil {
		fmt.Println(errorStyle.Render("FAILED"))
		results = append(results, DiagnosticResult{
			Name:    "Docker Version",
			Status:  "error",
			Message: "Cannot get Docker version",
			Details: err.Error(),
		})
	} else {
		fmt.Println(okStyle.Render(info.ServerVersion))
		results = append(results, DiagnosticResult{
			Name:    "Docker Version",
			Status:  "ok",
			Message: fmt.Sprintf("Docker %s", info.ServerVersion),
			Details: fmt.Sprintf("API: %s, OS: %s, Arch: %s", client.Client.ClientVersion(), info.OperatingSystem, info.Architecture),
		})
	}

	// Check 3: Docker daemon mode
	fmt.Print("Checking daemon mode... ")
	if info.Swarm.LocalNodeState == "active" {
		fmt.Println(warnStyle.Render("SWARM MODE"))
		results = append(results, DiagnosticResult{
			Name:    "Daemon Mode",
			Status:  "warn",
			Message: "Docker is running in Swarm mode",
			Details: "Some operations may behave differently in Swarm mode",
		})
	} else {
		fmt.Println(okStyle.Render("STANDALONE"))
		results = append(results, DiagnosticResult{
			Name:    "Daemon Mode",
			Status:  "ok",
			Message: "Docker is running in standalone mode",
		})
	}

	// Check 4: Storage driver
	fmt.Print("Checking storage driver... ")
	recommendedDrivers := map[string]bool{"overlay2": true, "btrfs": true}
	if recommendedDrivers[info.Driver] {
		fmt.Println(okStyle.Render(info.Driver))
		results = append(results, DiagnosticResult{
			Name:    "Storage Driver",
			Status:  "ok",
			Message: fmt.Sprintf("Using %s (recommended)", info.Driver),
		})
	} else {
		fmt.Println(warnStyle.Render(info.Driver))
		results = append(results, DiagnosticResult{
			Name:    "Storage Driver",
			Status:  "warn",
			Message: fmt.Sprintf("Using %s", info.Driver),
			Details: "Consider using overlay2 for better performance",
		})
	}

	// Check 5: Disk space
	fmt.Print("Checking disk usage... ")
	diskUsage, err := client.GetDiskUsage()
	if err != nil {
		fmt.Println(errorStyle.Render("FAILED"))
		results = append(results, DiagnosticResult{
			Name:    "Disk Usage",
			Status:  "error",
			Message: "Cannot get disk usage",
			Details: err.Error(),
		})
	} else {
		reclaimablePercent := float64(diskUsage.TotalReclaimable) / float64(diskUsage.Total) * 100
		if reclaimablePercent > 50 {
			fmt.Println(warnStyle.Render(fmt.Sprintf("%.0f%% reclaimable", reclaimablePercent)))
			results = append(results, DiagnosticResult{
				Name:    "Disk Usage",
				Status:  "warn",
				Message: fmt.Sprintf("%s used, %s reclaimable (%.0f%%)", humanize.Bytes(uint64(diskUsage.Total)), humanize.Bytes(uint64(diskUsage.TotalReclaimable)), reclaimablePercent),
				Details: "Consider running 'octo cleanup' or 'octo prune'",
			})
		} else {
			fmt.Println(okStyle.Render(humanize.Bytes(uint64(diskUsage.Total))))
			results = append(results, DiagnosticResult{
				Name:    "Disk Usage",
				Status:  "ok",
				Message: fmt.Sprintf("%s used, %s reclaimable", humanize.Bytes(uint64(diskUsage.Total)), humanize.Bytes(uint64(diskUsage.TotalReclaimable))),
			})
		}
	}

	// Check 6: Container count
	fmt.Print("Checking containers... ")
	containers, err := client.ListContainers(true)
	if err != nil {
		fmt.Println(errorStyle.Render("FAILED"))
	} else {
		running := 0
		stopped := 0
		for _, c := range containers {
			if c.State == "running" {
				running++
			} else {
				stopped++
			}
		}

		if stopped > 10 {
			fmt.Println(warnStyle.Render(fmt.Sprintf("%d stopped", stopped)))
			results = append(results, DiagnosticResult{
				Name:    "Containers",
				Status:  "warn",
				Message: fmt.Sprintf("%d running, %d stopped", running, stopped),
				Details: "Consider removing unused stopped containers with 'octo cleanup --containers'",
			})
		} else {
			fmt.Println(okStyle.Render(fmt.Sprintf("%d running", running)))
			results = append(results, DiagnosticResult{
				Name:    "Containers",
				Status:  "ok",
				Message: fmt.Sprintf("%d running, %d stopped", running, stopped),
			})
		}
	}

	// Check 7: Dangling images
	fmt.Print("Checking dangling images... ")
	danglingImages, err := client.GetDanglingImages()
	if err != nil {
		fmt.Println(errorStyle.Render("FAILED"))
	} else {
		if len(danglingImages) > 5 {
			var totalSize int64
			for _, img := range danglingImages {
				totalSize += img.Size
			}
			fmt.Println(warnStyle.Render(fmt.Sprintf("%d (%s)", len(danglingImages), humanize.Bytes(uint64(totalSize)))))
			results = append(results, DiagnosticResult{
				Name:    "Dangling Images",
				Status:  "warn",
				Message: fmt.Sprintf("%d dangling images (%s)", len(danglingImages), humanize.Bytes(uint64(totalSize))),
				Details: "Run 'octo cleanup --images' to remove dangling images",
			})
		} else {
			fmt.Println(okStyle.Render(fmt.Sprintf("%d", len(danglingImages))))
			results = append(results, DiagnosticResult{
				Name:    "Dangling Images",
				Status:  "ok",
				Message: fmt.Sprintf("%d dangling images", len(danglingImages)),
			})
		}
	}

	// Check 8: Unused volumes
	fmt.Print("Checking unused volumes... ")
	unusedVolumes, err := client.GetUnusedVolumes()
	if err != nil {
		fmt.Println(errorStyle.Render("FAILED"))
	} else {
		if len(unusedVolumes) > 5 {
			fmt.Println(warnStyle.Render(fmt.Sprintf("%d", len(unusedVolumes))))
			results = append(results, DiagnosticResult{
				Name:    "Unused Volumes",
				Status:  "warn",
				Message: fmt.Sprintf("%d unused volumes", len(unusedVolumes)),
				Details: "Run 'octo cleanup --volumes' to remove unused volumes",
			})
		} else {
			fmt.Println(okStyle.Render(fmt.Sprintf("%d", len(unusedVolumes))))
			results = append(results, DiagnosticResult{
				Name:    "Unused Volumes",
				Status:  "ok",
				Message: fmt.Sprintf("%d unused volumes", len(unusedVolumes)),
			})
		}
	}

	// Check 9: API response time
	fmt.Print("Checking API responsiveness... ")
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, err = client.Client.Ping(ctx)
	cancel()
	elapsed := time.Since(start)
	if err != nil {
		fmt.Println(errorStyle.Render("TIMEOUT"))
		results = append(results, DiagnosticResult{
			Name:    "API Response",
			Status:  "error",
			Message: "Docker API timed out",
			Details: "The Docker daemon may be overloaded or unresponsive",
		})
	} else if elapsed > 500*time.Millisecond {
		fmt.Println(warnStyle.Render(fmt.Sprintf("%dms", elapsed.Milliseconds())))
		results = append(results, DiagnosticResult{
			Name:    "API Response",
			Status:  "warn",
			Message: fmt.Sprintf("API response time: %dms", elapsed.Milliseconds()),
			Details: "Docker daemon may be under heavy load",
		})
	} else {
		fmt.Println(okStyle.Render(fmt.Sprintf("%dms", elapsed.Milliseconds())))
		results = append(results, DiagnosticResult{
			Name:    "API Response",
			Status:  "ok",
			Message: fmt.Sprintf("API response time: %dms", elapsed.Milliseconds()),
		})
	}

	// Check 10: Memory limits (Docker Desktop)
	fmt.Print("Checking memory configuration... ")
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		if info.MemTotal > 0 {
			memGB := float64(info.MemTotal) / (1024 * 1024 * 1024)
			if memGB < 2 {
				fmt.Println(warnStyle.Render(fmt.Sprintf("%.1fGB allocated", memGB)))
				results = append(results, DiagnosticResult{
					Name:    "Memory",
					Status:  "warn",
					Message: fmt.Sprintf("%.1fGB allocated to Docker", memGB),
					Details: "Consider increasing Docker Desktop memory allocation",
				})
			} else {
				fmt.Println(okStyle.Render(fmt.Sprintf("%.1fGB", memGB)))
				results = append(results, DiagnosticResult{
					Name:    "Memory",
					Status:  "ok",
					Message: fmt.Sprintf("%.1fGB allocated to Docker", memGB),
				})
			}
		} else {
			fmt.Println(infoStyle.Render("N/A"))
		}
	} else {
		fmt.Println(okStyle.Render("Native"))
		results = append(results, DiagnosticResult{
			Name:    "Memory",
			Status:  "ok",
			Message: "Running on native Linux (no VM overhead)",
		})
	}

	// Print summary
	printDiagnosticSummary(results, verbose)
}

func printDiagnosticSummary(results []DiagnosticResult, verbose bool) {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	fmt.Println()
	fmt.Println(strings.Repeat("─", 50))
	fmt.Println(titleStyle.Render("Summary"))
	fmt.Println()

	okCount := 0
	warnCount := 0
	errorCount := 0

	for _, r := range results {
		switch r.Status {
		case "ok":
			okCount++
		case "warn":
			warnCount++
		case "error":
			errorCount++
		}
	}

	fmt.Printf("  %s %d passed\n", okStyle.Render("✓"), okCount)
	if warnCount > 0 {
		fmt.Printf("  %s %d warnings\n", warnStyle.Render("!"), warnCount)
	}
	if errorCount > 0 {
		fmt.Printf("  %s %d errors\n", errorStyle.Render("✗"), errorCount)
	}

	// Show recommendations for warnings/errors
	hasRecommendations := false
	for _, r := range results {
		if (r.Status == "warn" || r.Status == "error") && r.Details != "" {
			if !hasRecommendations {
				fmt.Println()
				fmt.Println(titleStyle.Render("Recommendations"))
				hasRecommendations = true
			}
			icon := warnStyle.Render("!")
			if r.Status == "error" {
				icon = errorStyle.Render("✗")
			}
			fmt.Printf("\n  %s %s\n", icon, r.Name)
			fmt.Printf("    %s\n", infoStyle.Render(r.Details))
		}
	}

	// Verbose output
	if verbose {
		fmt.Println()
		fmt.Println(titleStyle.Render("Detailed Results"))
		for _, r := range results {
			icon := okStyle.Render("✓")
			if r.Status == "warn" {
				icon = warnStyle.Render("!")
			} else if r.Status == "error" {
				icon = errorStyle.Render("✗")
			}
			fmt.Printf("\n  %s %s: %s\n", icon, r.Name, r.Message)
			if r.Details != "" {
				fmt.Printf("    %s\n", infoStyle.Render(r.Details))
			}
		}
	}

	fmt.Println()
}
