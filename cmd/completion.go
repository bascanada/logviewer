// SPDX-License-Identifier: GPL-3.0-only
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `To load completions:

Bash:

  $ source <(logviewer completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ logviewer completion bash > /etc/bash_completion.d/logviewer
  # macOS:
  $ logviewer completion bash > $(brew --prefix)/etc/bash_completion.d/logviewer

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ logviewer completion zsh > "${fpath[1]}/_logviewer"

  # You will need to start a new shell for this setup to take effect.

Fish:

  $ logviewer completion fish | source

  # To load completions for each session, execute once:
  $ logviewer completion fish > ~/.config/fish/completions/logviewer.fish

PowerShell:

  PS> logviewer completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> logviewer completion powershell > logviewer.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
