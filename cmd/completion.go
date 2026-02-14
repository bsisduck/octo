package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish]",
	Short: "Generate shell completion scripts",
	Long: `Generate completion scripts for your shell.

To load completions:

Bash:
  $ source <(octo completion bash)
  # Or save to file:
  $ octo completion bash > /etc/bash_completion.d/octo

Zsh:
  $ octo completion zsh > "${fpath[1]}/_octo"
  # Then restart your shell or run: compinit

Fish:
  $ octo completion fish | source
  # Or save to file:
  $ octo completion fish > ~/.config/fish/completions/octo.fish`,
	ValidArgs:             []string{"bash", "zsh", "fish"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletionV2(os.Stdout, true)
		case "zsh":
			return cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		}
		return nil // unreachable due to ValidArgs + OnlyValidArgs
	},
}
