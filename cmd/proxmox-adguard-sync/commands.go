package main

import (
	"context"
	"errors"
	"fmt"
	"io"
)

var errCommandNotImplemented = errors.New(
	"command is not implemented yet",
)

type commandResult struct {
	Handled bool
	Err     error
}

func handleCommandLine(
	arguments []string,
	output io.Writer,
	errorOutput io.Writer,
) commandResult {
	if len(arguments) == 0 {
		return commandResult{}
	}

	if len(arguments) > 1 {
		err := errors.New("too many arguments")

		printCommandError(
			errorOutput,
			err.Error(),
		)

		printUsage(errorOutput)

		return commandResult{
			Handled: true,
			Err:     err,
		}
	}

	switch arguments[0] {
	case "help", "--help", "-h":
		printHelp(output)

		return commandResult{
			Handled: true,
		}

	case "--version", "-version", "version":
		printVersion(output)

		return commandResult{
			Handled: true,
		}

	case "setup":
		printCommandError(
			errorOutput,
			"the setup wizard is not implemented yet",
		)

		return commandResult{
			Handled: true,
			Err: fmt.Errorf(
				"setup: %w",
				errCommandNotImplemented,
			),
		}

	case "validate":
		err := runValidation(
			context.Background(),
			output,
		)
		if err != nil {
			printCommandError(
				errorOutput,
				fmt.Sprintf(
					"validation failed: %v",
					err,
				),
			)
		}

		return commandResult{
			Handled: true,
			Err:     err,
		}

	default:
		err := fmt.Errorf(
			"unknown command %q",
			arguments[0],
		)

		printCommandError(
			errorOutput,
			err.Error(),
		)

		printUsage(errorOutput)

		return commandResult{
			Handled: true,
			Err:     err,
		}
	}
}

func printHelp(writer io.Writer) {
	_, _ = fmt.Fprintf(
		writer,
		`%s synchronizes Proxmox VE guest addresses with AdGuard Home DNS rewrites.

Usage:
  %s
  %s <command>

Commands:
  help       Show this help information
  setup      Create a configuration interactively
  validate   Validate configuration and service connectivity
  version    Show version and build information

Options:
  -h, --help       Show this help information
      --version    Show version and build information

Running %s without a command starts the synchronization service.

The setup command is planned but is not implemented yet.
`,
		applicationName,
		applicationName,
		applicationName,
		applicationName,
	)
}

func printUsage(writer io.Writer) {
	_, _ = fmt.Fprintf(
		writer,
		"Usage: %s [help|setup|validate|version]\n",
		applicationName,
	)
}

func printCommandError(
	writer io.Writer,
	message string,
) {
	_, _ = fmt.Fprintf(
		writer,
		"%s: %s\n",
		applicationName,
		message,
	)
}
