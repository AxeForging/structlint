package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

// NewCompletionCmd provides shell completion scripts.
func NewCompletionCmd() *cli.Command {
	return &cli.Command{
		Name:  "completion",
		Usage: "generate shell completion scripts",
		Commands: []*cli.Command{
			BashCompleteCommand(),
			ZshCompleteCommand(),
			FishCompleteCommand(),
		},
	}
}

func BashCompleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "bash",
		Usage: "print bash completion script",
		Action: func(ctx context.Context, c *cli.Command) error {
			fmt.Print(bashCompletion)
			return nil
		},
	}
}

func ZshCompleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "zsh",
		Usage: "print zsh completion script",
		Action: func(ctx context.Context, c *cli.Command) error {
			fmt.Print(zshCompletion)
			return nil
		},
	}
}

func FishCompleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "fish",
		Usage: "print fish completion script",
		Action: func(ctx context.Context, c *cli.Command) error {
			fmt.Print(fishCompletion)
			return nil
		},
	}
}

const bashCompletion = `# bash completion for structlint
# Add to ~/.bashrc: eval "$(structlint completion bash)"

_structlint_completions() {
    local cur prev commands global_flags
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    commands="validate init version completion"
    global_flags="--config --log-level --no-color --json-output --silent --help"

    case "${prev}" in
        structlint)
            COMPREPLY=( $(compgen -W "${commands} ${global_flags}" -- "${cur}") )
            return 0
            ;;
        validate)
            COMPREPLY=( $(compgen -W "--path --json-output --silent --group-violations --verbose --help" -- "${cur}") )
            return 0
            ;;
        init)
            COMPREPLY=( $(compgen -W "--type --force --help" -- "${cur}") )
            return 0
            ;;
        --type)
            COMPREPLY=( $(compgen -W "go node python generic" -- "${cur}") )
            return 0
            ;;
        --config|--path|--json-output)
            COMPREPLY=( $(compgen -f -- "${cur}") )
            return 0
            ;;
        --log-level)
            COMPREPLY=( $(compgen -W "debug info warn error" -- "${cur}") )
            return 0
            ;;
        completion)
            COMPREPLY=( $(compgen -W "bash zsh fish" -- "${cur}") )
            return 0
            ;;
    esac

    COMPREPLY=( $(compgen -W "${commands} ${global_flags}" -- "${cur}") )
}

complete -F _structlint_completions structlint
`

const zshCompletion = `# zsh completion for structlint
# Add to ~/.zshrc: eval "$(structlint completion zsh)"

_structlint() {
    local -a commands
    commands=(
        'validate:validate directory structure and file naming patterns'
        'init:generate a starter .structlint.yaml configuration file'
        'version:print version information'
        'completion:generate shell completion scripts'
    )

    _arguments -C \
        '--config[path to the configuration file]:file:_files' \
        '--log-level[logging level]:level:(debug info warn error)' \
        '--no-color[disable colored output]' \
        '--json-output[path to save the JSON report]:file:_files' \
        '--silent[suppress all output except for the JSON report]' \
        '--help[show help]' \
        '1:command:->cmds' \
        '*::arg:->args'

    case "$state" in
        cmds)
            _describe -t commands 'structlint command' commands
            ;;
        args)
            case "${words[1]}" in
                validate)
                    _arguments \
                        '--path[path to validate]:directory:_directories' \
                        '--json-output[path to save the JSON report]:file:_files' \
                        '--silent[suppress all output]' \
                        '--group-violations[group violations by type]' \
                        '--verbose[show all allowed files]' \
                        '--help[show help]'
                    ;;
                init)
                    _arguments \
                        '--type[project type]:type:(go node python generic)' \
                        '--force[overwrite existing configuration]' \
                        '--help[show help]'
                    ;;
                completion)
                    _arguments '1:shell:(bash zsh fish)'
                    ;;
            esac
            ;;
    esac
}

compdef _structlint structlint
`

const fishCompletion = `# fish completion for structlint
# Add to ~/.config/fish/completions/structlint.fish

# Disable file completions by default
complete -c structlint -f

# Commands
complete -c structlint -n '__fish_use_subcommand' -a 'validate' -d 'Validate directory structure and file naming patterns'
complete -c structlint -n '__fish_use_subcommand' -a 'init' -d 'Generate a starter .structlint.yaml configuration file'
complete -c structlint -n '__fish_use_subcommand' -a 'version' -d 'Print version information'
complete -c structlint -n '__fish_use_subcommand' -a 'completion' -d 'Generate shell completion scripts'

# Global flags
complete -c structlint -l config -d 'Path to the configuration file' -r -F
complete -c structlint -l log-level -d 'Logging level' -r -a 'debug info warn error'
complete -c structlint -l no-color -d 'Disable colored output'
complete -c structlint -l json-output -d 'Path to save the JSON report' -r -F
complete -c structlint -l silent -d 'Suppress all output except for the JSON report'

# validate flags
complete -c structlint -n '__fish_seen_subcommand_from validate' -l path -d 'Path to validate' -r -F
complete -c structlint -n '__fish_seen_subcommand_from validate' -l json-output -d 'Path to save the JSON report' -r -F
complete -c structlint -n '__fish_seen_subcommand_from validate' -l silent -d 'Suppress all output'
complete -c structlint -n '__fish_seen_subcommand_from validate' -l group-violations -d 'Group violations by type'
complete -c structlint -n '__fish_seen_subcommand_from validate' -l verbose -d 'Show all allowed files'

# init flags
complete -c structlint -n '__fish_seen_subcommand_from init' -l type -d 'Project type' -r -a 'go node python generic'
complete -c structlint -n '__fish_seen_subcommand_from init' -l force -d 'Overwrite existing configuration'

# completion subcommands
complete -c structlint -n '__fish_seen_subcommand_from completion' -a 'bash' -d 'Print bash completion script'
complete -c structlint -n '__fish_seen_subcommand_from completion' -a 'zsh' -d 'Print zsh completion script'
complete -c structlint -n '__fish_seen_subcommand_from completion' -a 'fish' -d 'Print fish completion script'
`
