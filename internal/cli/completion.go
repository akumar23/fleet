package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// newCompletionCmd creates the completion command for generating shell completions
func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for Fleet CLI.

The completion script must be sourced to provide completions. After generating the
completion script, follow the instructions for your shell:

Bash:
  $ source <(fleet completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ fleet completion bash > /etc/bash_completion.d/fleet
  # macOS:
  $ fleet completion bash > $(brew --prefix)/etc/bash_completion.d/fleet

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ fleet completion zsh > "${fpath[1]}/_fleet"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ fleet completion fish | source

  # To load completions for each session, execute once:
  $ fleet completion fish > ~/.config/fish/completions/fleet.fish

PowerShell:
  PS> fleet completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> fleet completion powershell > fleet.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		// Skip parent's PersistentPreRunE (config loading) for completion command
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCompletion(cmd, args[0])
		},
	}

	return cmd
}

// runCompletion generates the completion script for the specified shell
func runCompletion(cmd *cobra.Command, shell string) error {
	switch shell {
	case "bash":
		return cmd.Root().GenBashCompletion(os.Stdout)
	case "zsh":
		return cmd.Root().GenZshCompletion(os.Stdout)
	case "fish":
		return cmd.Root().GenFishCompletion(os.Stdout, true)
	case "powershell":
		return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
	default:
		return fmt.Errorf("unsupported shell type %q", shell)
	}
}
