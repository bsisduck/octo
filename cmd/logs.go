package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/bsisduck/octo/internal/docker"
	"github.com/bsisduck/octo/internal/ui/format"
)

// LogOutputEntry is used for JSON/YAML log output
type LogOutputEntry struct {
	Timestamp string `json:"timestamp" yaml:"timestamp"`
	Stream    string `json:"stream" yaml:"stream"`
	Content   string `json:"content" yaml:"content"`
}

var logsCmd = &cobra.Command{
	Use:   "logs <container-id>",
	Short: "View container logs",
	Long: `View logs from a Docker container.

Examples:
  octo logs my-container
  octo logs my-container --tail 50
  octo logs my-container --follow
  octo logs my-container --output-format json`,
	Args: cobra.ExactArgs(1),
	RunE: runLogs,
}

func init() {
	logsCmd.Flags().IntP("tail", "n", 100, "Number of lines to show from end of logs")
	logsCmd.Flags().BoolP("follow", "f", false, "Follow log output")
}

func runLogs(cmd *cobra.Command, args []string) error {
	containerID := args[0]
	tail, _ := cmd.Flags().GetInt("tail")
	follow, _ := cmd.Flags().GetBool("follow")
	outputFormat, _ := cmd.Flags().GetString("output-format")

	client, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("connecting to Docker: %w", err)
	}
	defer client.Close()

	// Fetch initial logs
	ctx := context.Background()
	entries, err := client.GetContainerLogs(ctx, containerID, tail)
	if err != nil {
		return fmt.Errorf("fetching logs: %w", err)
	}

	switch outputFormat {
	case "json":
		output := make([]LogOutputEntry, len(entries))
		for i, e := range entries {
			output[i] = LogOutputEntry{
				Timestamp: e.Timestamp.Format("2006-01-02T15:04:05.000Z"),
				Stream:    e.Stream,
				Content:   e.Content,
			}
		}
		return format.FormatJSON(os.Stdout, output)
	case "yaml":
		output := make([]LogOutputEntry, len(entries))
		for i, e := range entries {
			output[i] = LogOutputEntry{
				Timestamp: e.Timestamp.Format("2006-01-02T15:04:05.000Z"),
				Stream:    e.Stream,
				Content:   e.Content,
			}
		}
		return format.FormatYAML(os.Stdout, output)
	}

	// Text output
	for _, e := range entries {
		ts := e.Timestamp.Format("2006-01-02 15:04:05")
		fmt.Printf("%s  %-6s  %s\n", ts, e.Stream, e.Content)
	}

	if !follow {
		return nil
	}

	// Follow mode: stream new logs
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	logCh, errCh, cancel := client.StreamContainerLogs(ctx, containerID)
	defer cancel()

	for {
		select {
		case entry, ok := <-logCh:
			if !ok {
				return nil
			}
			ts := entry.Timestamp.Format("2006-01-02 15:04:05")
			fmt.Printf("%s  %-6s  %s\n", ts, entry.Stream, entry.Content)
		case err := <-errCh:
			if err != nil {
				return fmt.Errorf("log stream error: %w", err)
			}
			return nil
		case <-sigCh:
			return nil
		}
	}
}
