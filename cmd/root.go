// Package cmd provides the CLI command structure for Octo.
package cmd

import (
	"fmt"
	"os"

	"github.com/bsisduck/octo/internal/ui/styles"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		// Launch interactive menu when no subcommand is provided
		return runInteractiveMenu()
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Validate output-format flag
		outputFormat, _ := cmd.Flags().GetString("output-format")
		switch outputFormat {
		case "text", "json", "yaml":
			// Valid formats
		default:
			return fmt.Errorf("invalid output format: %s. Choose: text, json, yaml", outputFormat)
		}

		if noColor || os.Getenv("NO_COLOR") != "" {
			styles.DisableColors()
		}
		return nil
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
	rootCmd.PersistentFlags().String("output-format", "text", "Output format: text, json, yaml")

	// Register completion for output-format flag
	rootCmd.RegisterFlagCompletionFunc("output-format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"text", "json", "yaml"}, cobra.ShellCompDirectiveDefault
	})

	// Add subcommands
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(cleanupCmd)
	rootCmd.AddCommand(pruneCmd)
	rootCmd.AddCommand(diagnoseCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(logsCmd)
}

// runInteractiveMenu launches the TUI-based interactive menu
// and dispatches the selected command after the TUI exits.
func runInteractiveMenu() error {
	menu := NewInteractiveMenu()
	action, err := menu.Run()
	if err != nil {
		return fmt.Errorf("menu error: %w", err)
	}

	if action == "" {
		return nil // user quit without selecting (q/esc/ctrl+c)
	}

	// Dispatch to the appropriate command
	switch action {
	case "status":
		return statusCmd.RunE(statusCmd, nil)
	case "analyze":
		return analyzeCmd.RunE(analyzeCmd, nil)
	case "cleanup":
		return cleanupCmd.RunE(cleanupCmd, nil)
	case "prune":
		return pruneCmd.RunE(pruneCmd, nil)
	case "diagnose":
		return diagnoseCmd.RunE(diagnoseCmd, nil)
	case "version":
		versionCmd.Run(versionCmd, nil) // version is safe/simple, kept as Run for now or update later
		return nil
	default:
		return fmt.Errorf("unknown action: %s", action)
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
