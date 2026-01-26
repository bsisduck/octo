// Package cmd provides the CLI command structure for Octo.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version information
	Version   = "0.1.0"
	BuildTime = ""
	GitCommit = ""

	// Global flags
	debug   bool
	dryRun  bool
	noColor bool
)

const (
	octoTagline = "Orchestrate your Docker containers like an octopus."
	octoLogo    = `
   ___       _
  / _ \  ___| |_ ___
 | | | |/ __| __/ _ \
 | |_| | (__| || (_) |
  \___/ \___|\__\___/
`
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "octo",
	Short: "Docker container management CLI",
	Long: fmt.Sprintf(`%s
%s

Octo helps you manage Docker containers, images, volumes, and networks
with an intuitive interface and powerful cleanup capabilities.

Run 'octo' without arguments to launch the interactive menu.`, octoLogo, octoTagline),
	Run: func(cmd *cobra.Command, args []string) {
		// Launch interactive menu when no subcommand is provided
		runInteractiveMenu()
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags available to all commands
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug output")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Preview changes without executing")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")

	// Add subcommands
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(cleanupCmd)
	rootCmd.AddCommand(pruneCmd)
	rootCmd.AddCommand(diagnoseCmd)
	rootCmd.AddCommand(versionCmd)
}

// runInteractiveMenu launches the TUI-based interactive menu
func runInteractiveMenu() {
	menu := NewInteractiveMenu()
	if err := menu.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// IsDebug returns whether debug mode is enabled
func IsDebug() bool {
	return debug || os.Getenv("OCTO_DEBUG") == "1"
}

// IsDryRun returns whether dry-run mode is enabled
func IsDryRun() bool {
	return dryRun || os.Getenv("OCTO_DRY_RUN") == "1"
}

// NoColor returns whether color output is disabled
func NoColor() bool {
	return noColor || os.Getenv("NO_COLOR") != ""
}
