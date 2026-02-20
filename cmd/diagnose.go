package cmd

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"

	"github.com/bsisduck/octo/internal/docker"
	"github.com/bsisduck/octo/internal/ui/format"
	"github.com/bsisduck/octo/internal/ui/styles"
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
	RunE: runDiagnose,
}

func init() {
	diagnoseCmd.Flags().Bool("verbose", false, "Show detailed diagnostic information")
}

type DiagnosticResult struct {
	Name    string `json:"name" yaml:"name"`
	Status  string `json:"status" yaml:"status"`
	Message string `json:"message" yaml:"message"`
	Details string `json:"details,omitempty" yaml:"details,omitempty"`
}

// DiagnoseOutput holds structured diagnostic data for JSON/YAML output
type DiagnoseOutput struct {
	Results []DiagnosticResult `json:"results" yaml:"results"`
	Summary DiagnoseSummary    `json:"summary" yaml:"summary"`
}

// DiagnoseSummary holds pass/warn/error counts
type DiagnoseSummary struct {
	Passed   int `json:"passed" yaml:"passed"`
	Warnings int `json:"warnings" yaml:"warnings"`
	Errors   int `json:"errors" yaml:"errors"`
}

func runDiagnose(cmd *cobra.Command, args []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")
	outputFormat, _ := cmd.Flags().GetString("output-format")
	textMode := outputFormat != "json" && outputFormat != "yaml"

	// Styles (defined in internal/ui/styles/theme.go)
	titleStyle := styles.Title
	okStyle := styles.Success
	warnStyle := styles.Warning
	errorStyle := styles.Error
	infoStyle := styles.Info

	if textMode {
		fmt.Println()
		fmt.Println(titleStyle.Render("🐙 Octo Docker Diagnostics"))
		fmt.Println(strings.Repeat("─", 50))
		fmt.Println()
	}

	results := []DiagnosticResult{}
	ctx := context.Background()

	// Check 1: Docker connection
	if textMode {
		fmt.Print("Checking Docker connection... ")
	}
	client, err := docker.NewClient()
	if err != nil {
		if textMode {
			fmt.Println(errorStyle.Render("FAILED"))
		}
		results = append(results, DiagnosticResult{
			Name:    "Docker Connection",
			Status:  "error",
			Message: "Cannot connect to Docker daemon",
			Details: err.Error(),
		})
		if !textMode {
			return outputDiagnoseStructured(outputFormat, results)
		}
		printDiagnosticSummary(results, verbose)
		return fmt.Errorf("docker connection failed: %w", err)
	}
	if textMode {
		fmt.Println(okStyle.Render("OK"))
	}
	results = append(results, DiagnosticResult{
		Name:    "Docker Connection",
		Status:  "ok",
		Message: "Connected to Docker daemon",
	})
	defer client.Close()

	// Check 2: Docker version
	if textMode {
		fmt.Print("Checking Docker version... ")
	}
	info, err := client.GetServerInfo(ctx)
	if err != nil {
		if textMode {
			fmt.Println(errorStyle.Render("FAILED"))
		}
		results = append(results, DiagnosticResult{
			Name:    "Docker Version",
			Status:  "error",
			Message: "Cannot get Docker version",
			Details: err.Error(),
		})
	} else {
		if textMode {
			fmt.Println(okStyle.Render(info.ServerVersion))
		}
		results = append(results, DiagnosticResult{
			Name:    "Docker Version",
			Status:  "ok",
			Message: fmt.Sprintf("Docker %s", info.ServerVersion),
			Details: fmt.Sprintf("OS: %s, Arch: %s", info.OperatingSystem, info.Architecture),
		})
	}

	// Check 3: Docker daemon mode
	if textMode {
		fmt.Print("Checking daemon mode... ")
	}
	if info.Swarm.LocalNodeState == "active" {
		if textMode {
			fmt.Println(warnStyle.Render("SWARM MODE"))
		}
		results = append(results, DiagnosticResult{
			Name:    "Daemon Mode",
			Status:  "warn",
			Message: "Docker is running in Swarm mode",
			Details: "Some operations may behave differently in Swarm mode",
		})
	} else {
		if textMode {
			fmt.Println(okStyle.Render("STANDALONE"))
		}
		results = append(results, DiagnosticResult{
			Name:    "Daemon Mode",
			Status:  "ok",
			Message: "Docker is running in standalone mode",
		})
	}

	// Check 4: Storage driver
	if textMode {
		fmt.Print("Checking storage driver... ")
	}
	recommendedDrivers := map[string]bool{"overlay2": true, "btrfs": true}
	if recommendedDrivers[info.Driver] {
		if textMode {
			fmt.Println(okStyle.Render(info.Driver))
		}
		results = append(results, DiagnosticResult{
			Name:    "Storage Driver",
			Status:  "ok",
			Message: fmt.Sprintf("Using %s (recommended)", info.Driver),
		})
	} else {
		if textMode {
			fmt.Println(warnStyle.Render(info.Driver))
		}
		results = append(results, DiagnosticResult{
			Name:    "Storage Driver",
			Status:  "warn",
			Message: fmt.Sprintf("Using %s", info.Driver),
			Details: "Consider using overlay2 for better performance",
		})
	}

	// Check 5: Disk space
	if textMode {
		fmt.Print("Checking disk usage... ")
	}
	diskUsage, err := client.GetDiskUsage(ctx)
	if err != nil {
		if textMode {
			fmt.Println(errorStyle.Render("FAILED"))
		}
		results = append(results, DiagnosticResult{
			Name:    "Disk Usage",
			Status:  "error",
			Message: "Cannot get disk usage",
			Details: err.Error(),
		})
	} else {
		var reclaimablePercent float64
		if diskUsage.Total > 0 {
			reclaimablePercent = float64(diskUsage.TotalReclaimable) / float64(diskUsage.Total) * 100
		}
		if reclaimablePercent > 50 {
			if textMode {
				fmt.Println(warnStyle.Render(fmt.Sprintf("%.0f%% reclaimable", reclaimablePercent)))
			}
			results = append(results, DiagnosticResult{
				Name:    "Disk Usage",
				Status:  "warn",
				Message: fmt.Sprintf("%s used, %s reclaimable (%.0f%%)", humanize.Bytes(uint64(diskUsage.Total)), humanize.Bytes(uint64(diskUsage.TotalReclaimable)), reclaimablePercent),
				Details: "Consider running 'octo cleanup' or 'octo prune'",
			})
		} else {
			if textMode {
				fmt.Println(okStyle.Render(humanize.Bytes(uint64(diskUsage.Total))))
			}
			results = append(results, DiagnosticResult{
				Name:    "Disk Usage",
				Status:  "ok",
				Message: fmt.Sprintf("%s used, %s reclaimable", humanize.Bytes(uint64(diskUsage.Total)), humanize.Bytes(uint64(diskUsage.TotalReclaimable))),
			})
		}
	}

	// Check 6: Container count
	if textMode {
		fmt.Print("Checking containers... ")
	}
	containers, err := client.ListContainers(ctx, true)
	if err != nil {
		if textMode {
			fmt.Println(errorStyle.Render("FAILED"))
		}
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
			if textMode {
				fmt.Println(warnStyle.Render(fmt.Sprintf("%d stopped", stopped)))
			}
			results = append(results, DiagnosticResult{
				Name:    "Containers",
				Status:  "warn",
				Message: fmt.Sprintf("%d running, %d stopped", running, stopped),
				Details: "Consider removing unused stopped containers with 'octo cleanup --containers'",
			})
		} else {
			if textMode {
				fmt.Println(okStyle.Render(fmt.Sprintf("%d running", running)))
			}
			results = append(results, DiagnosticResult{
				Name:    "Containers",
				Status:  "ok",
				Message: fmt.Sprintf("%d running, %d stopped", running, stopped),
			})
		}
	}

	// Check 7: Dangling images
	if textMode {
		fmt.Print("Checking dangling images... ")
	}
	danglingImages, err := client.GetDanglingImages(ctx)
	if err != nil {
		if textMode {
			fmt.Println(errorStyle.Render("FAILED"))
		}
	} else {
		if len(danglingImages) > 5 {
			var totalSize int64
			for _, img := range danglingImages {
				totalSize += img.Size
			}
			if textMode {
				fmt.Println(warnStyle.Render(fmt.Sprintf("%d (%s)", len(danglingImages), humanize.Bytes(uint64(totalSize)))))
			}
			results = append(results, DiagnosticResult{
				Name:    "Dangling Images",
				Status:  "warn",
				Message: fmt.Sprintf("%d dangling images (%s)", len(danglingImages), humanize.Bytes(uint64(totalSize))),
				Details: "Run 'octo cleanup --images' to remove dangling images",
			})
		} else {
			if textMode {
				fmt.Println(okStyle.Render(fmt.Sprintf("%d", len(danglingImages))))
			}
			results = append(results, DiagnosticResult{
				Name:    "Dangling Images",
				Status:  "ok",
				Message: fmt.Sprintf("%d dangling images", len(danglingImages)),
			})
		}
	}

	// Check 8: Unused volumes
	if textMode {
		fmt.Print("Checking unused volumes... ")
	}
	unusedVolumes, err := client.GetUnusedVolumes(ctx)
	if err != nil {
		if textMode {
			fmt.Println(errorStyle.Render("FAILED"))
		}
	} else {
		if len(unusedVolumes) > 5 {
			if textMode {
				fmt.Println(warnStyle.Render(fmt.Sprintf("%d", len(unusedVolumes))))
			}
			results = append(results, DiagnosticResult{
				Name:    "Unused Volumes",
				Status:  "warn",
				Message: fmt.Sprintf("%d unused volumes", len(unusedVolumes)),
				Details: "Run 'octo cleanup --volumes' to remove unused volumes",
			})
		} else {
			if textMode {
				fmt.Println(okStyle.Render(fmt.Sprintf("%d", len(unusedVolumes))))
			}
			results = append(results, DiagnosticResult{
				Name:    "Unused Volumes",
				Status:  "ok",
				Message: fmt.Sprintf("%d unused volumes", len(unusedVolumes)),
			})
		}
	}

	// Check 9: API response time
	if textMode {
		fmt.Print("Checking API responsiveness... ")
	}
	start := time.Now()
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = client.Ping(pingCtx)
	pingCancel()
	elapsed := time.Since(start)
	if err != nil {
		if textMode {
			fmt.Println(errorStyle.Render("TIMEOUT"))
		}
		results = append(results, DiagnosticResult{
			Name:    "API Response",
			Status:  "error",
			Message: "Docker API timed out",
			Details: "The Docker daemon may be overloaded or unresponsive",
		})
	} else if elapsed > 500*time.Millisecond {
		if textMode {
			fmt.Println(warnStyle.Render(fmt.Sprintf("%dms", elapsed.Milliseconds())))
		}
		results = append(results, DiagnosticResult{
			Name:    "API Response",
			Status:  "warn",
			Message: fmt.Sprintf("API response time: %dms", elapsed.Milliseconds()),
			Details: "Docker daemon may be under heavy load",
		})
	} else {
		if textMode {
			fmt.Println(okStyle.Render(fmt.Sprintf("%dms", elapsed.Milliseconds())))
		}
		results = append(results, DiagnosticResult{
			Name:    "API Response",
			Status:  "ok",
			Message: fmt.Sprintf("API response time: %dms", elapsed.Milliseconds()),
		})
	}

	// Check 10: Memory limits (Docker Desktop)
	if textMode {
		fmt.Print("Checking memory configuration... ")
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		if info.MemTotal > 0 {
			memGB := float64(info.MemTotal) / (1024 * 1024 * 1024)
			if memGB < 2 {
				if textMode {
					fmt.Println(warnStyle.Render(fmt.Sprintf("%.1fGB allocated", memGB)))
				}
				results = append(results, DiagnosticResult{
					Name:    "Memory",
					Status:  "warn",
					Message: fmt.Sprintf("%.1fGB allocated to Docker", memGB),
					Details: "Consider increasing Docker Desktop memory allocation",
				})
			} else {
				if textMode {
					fmt.Println(okStyle.Render(fmt.Sprintf("%.1fGB", memGB)))
				}
				results = append(results, DiagnosticResult{
					Name:    "Memory",
					Status:  "ok",
					Message: fmt.Sprintf("%.1fGB allocated to Docker", memGB),
				})
			}
		} else {
			if textMode {
				fmt.Println(infoStyle.Render("N/A"))
			}
		}
	} else {
		if textMode {
			fmt.Println(okStyle.Render("Native"))
		}
		results = append(results, DiagnosticResult{
			Name:    "Memory",
			Status:  "ok",
			Message: "Running on native Linux (no VM overhead)",
		})
	}

	// Structured output for JSON/YAML
	if !textMode {
		return outputDiagnoseStructured(outputFormat, results)
	}

	// Print summary
	printDiagnosticSummary(results, verbose)
	return nil
}

func outputDiagnoseStructured(outputFormat string, results []DiagnosticResult) error {
	passed, warnings, errors := 0, 0, 0
	for _, r := range results {
		switch r.Status {
		case "ok":
			passed++
		case "warn":
			warnings++
		case "error":
			errors++
		}
	}
	output := DiagnoseOutput{
		Results: results,
		Summary: DiagnoseSummary{
			Passed:   passed,
			Warnings: warnings,
			Errors:   errors,
		},
	}
	switch outputFormat {
	case "json":
		return format.FormatJSON(os.Stdout, output)
	case "yaml":
		return format.FormatYAML(os.Stdout, output)
	}
	return nil
}

func printDiagnosticSummary(results []DiagnosticResult, verbose bool) {
	// Styles (defined in internal/ui/styles/theme.go)
	titleStyle := styles.Title
	okStyle := styles.Success
	warnStyle := styles.Warning
	errorStyle := styles.Error
	infoStyle := styles.Info

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
