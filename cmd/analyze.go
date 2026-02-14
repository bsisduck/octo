package cmd

import (
	"fmt"

	"github.com/bsisduck/octo/internal/docker"
	"github.com/bsisduck/octo/internal/tui/analyze"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

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

	client, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("connecting to Docker: %w", err)
	}
	defer client.Close()

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
