package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionExample = `
	# Installing bash completion on macOS using homebrew
	## If running Bash 3.2 included with macOS
	brew install bash-completion
	## or, if running Bash 4.1+
	brew install bash-completion@2
	## Add the completion to your completion directory
	dapr completion bash > $(brew --prefix)/etc/bash_completion.d/dapr
	source ~/.bash_profile

	# Installing bash completion on Linux
	## If bash-completion is not installed on Linux, please install the 'bash-completion' package
	## via your distribution's package manager.
	## Load the dapr completion code for bash into the current shell
	source <(dapr completion bash)
	## Write bash completion code to a file and source if from .bash_profile
	dapr completion bash > ~/.dapr/completion.bash.inc
	printf "
	## dapr shell completion
	source '$HOME/.dapr/completion.bash.inc'
	" >> $HOME/.bash_profile
	source $HOME/.bash_profile

	# Installing zsh completion on macOS using homebrew
	## If zsh-completion is not installed on macOS, please install the 'zsh-completion' package
	brew install zsh-completions
	## Set the dapr completion code for zsh[1] to autoload on startup
	dapr completion zsh > "${fpath[1]}/_dapr"
	source ~/.zshrc

	# Installing zsh completion on Linux
	## If zsh-completion is not installed on Linux, please install the 'zsh-completion' package
	## via your distribution's package manager.
	## Load the dapr completion code for zsh into the current shell
	source <(dapr completion zsh)
	# Set the dapr completion code for zsh[1] to autoload on startup
  	dapr completion zsh > "${fpath[1]}/_dapr"

	# Installing powershell completion on Windows
	## Create $PROFILE if it not exists
	if (!(Test-Path -Path $PROFILE )){ New-Item -Type File -Path $PROFILE -Force }
	## Add the completion to your profile
	dapr completion powershell >> $PROFILE
`

func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "completion",
		Short:   "Generates shell completion scripts",
		Example: completionExample,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	cmd.AddCommand(
		newCompletionBashCmd(),
		newCompletionZshCmd(),
		newCompletionPowerShellCmd(),
	)

	return cmd
}

func newCompletionBashCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bash",
		Short: "Generates bash completion scripts",
		Run: func(cmd *cobra.Command, args []string) {
			RootCmd.GenBashCompletion(os.Stdout)
		},
	}

	return cmd
}

func newCompletionZshCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "zsh",
		Short: "Generates zsh completion scripts",
		Run: func(cmd *cobra.Command, args []string) {
			RootCmd.GenZshCompletion(os.Stdout)
		},
	}

	return cmd
}

func newCompletionPowerShellCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "powershell",
		Short: "Generates powershell completion scripts",
		Run: func(cmd *cobra.Command, args []string) {
			RootCmd.GenPowerShellCompletion(os.Stdout)
		},
	}

	return cmd
}

func init() {
	RootCmd.AddCommand(newCompletionCmd())
}
