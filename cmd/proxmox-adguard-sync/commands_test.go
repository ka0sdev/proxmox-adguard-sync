package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/ka0sdev/proxmox-adguard-sync/internal/config"
)

func TestHandleCommandLineWithoutArguments(
	t *testing.T,
) {
	var output bytes.Buffer
	var errorOutput bytes.Buffer

	result := handleCommandLine(
		nil,
		&output,
		&errorOutput,
	)

	if result.Handled {
		t.Fatal(
			"handleCommandLine() returned Handled=true",
		)
	}

	if result.Err != nil {
		t.Fatalf(
			"handleCommandLine() error = %v",
			result.Err,
		)
	}

	if output.Len() != 0 {
		t.Errorf(
			"output = %q, expected no output",
			output.String(),
		)
	}

	if errorOutput.Len() != 0 {
		t.Errorf(
			"error output = %q, expected no output",
			errorOutput.String(),
		)
	}
}

func TestHandleCommandLineVersion(
	t *testing.T,
) {
	arguments := []string{
		"--version",
		"-version",
		"version",
	}

	for _, argument := range arguments {
		t.Run(
			argument,
			func(t *testing.T) {
				var output bytes.Buffer
				var errorOutput bytes.Buffer

				result := handleCommandLine(
					[]string{argument},
					&output,
					&errorOutput,
				)

				if !result.Handled {
					t.Fatal(
						"handleCommandLine() returned Handled=false",
					)
				}

				if result.Err != nil {
					t.Fatalf(
						"handleCommandLine() error = %v",
						result.Err,
					)
				}

				if !strings.Contains(
					output.String(),
					applicationName,
				) {
					t.Errorf(
						"output = %q, expected application name",
						output.String(),
					)
				}

				if errorOutput.Len() != 0 {
					t.Errorf(
						"error output = %q, expected no output",
						errorOutput.String(),
					)
				}
			},
		)
	}
}

func TestHandleCommandLineHelp(
	t *testing.T,
) {
	arguments := []string{
		"help",
		"--help",
		"-h",
	}

	for _, argument := range arguments {
		t.Run(
			argument,
			func(t *testing.T) {
				var output bytes.Buffer
				var errorOutput bytes.Buffer

				result := handleCommandLine(
					[]string{argument},
					&output,
					&errorOutput,
				)

				if !result.Handled {
					t.Fatal(
						"handleCommandLine() returned Handled=false",
					)
				}

				if result.Err != nil {
					t.Fatalf(
						"handleCommandLine() error = %v",
						result.Err,
					)
				}

				if !strings.Contains(
					output.String(),
					"Usage:",
				) {
					t.Errorf(
						"output = %q, expected usage information",
						output.String(),
					)
				}

				if errorOutput.Len() != 0 {
					t.Errorf(
						"error output = %q, expected no output",
						errorOutput.String(),
					)
				}
			},
		)
	}
}

func TestHandleCommandLineSetup(
	t *testing.T,
) {
	originalRunSetupCommand := runSetupCommand

	t.Cleanup(
		func() {
			runSetupCommand =
				originalRunSetupCommand
		},
	)

	runSetupCommand = func(
		output io.Writer,
		errorOutput io.Writer,
	) error {
		_, _ = fmt.Fprintln(
			output,
			"setup completed",
		)

		return nil
	}

	var output bytes.Buffer
	var errorOutput bytes.Buffer

	result := handleCommandLine(
		[]string{"setup"},
		&output,
		&errorOutput,
	)

	if !result.Handled {
		t.Fatal(
			"handleCommandLine() returned Handled=false",
		)
	}

	if result.Err != nil {
		t.Fatalf(
			"handleCommandLine() error = %v",
			result.Err,
		)
	}

	if !strings.Contains(
		output.String(),
		"setup completed",
	) {
		t.Errorf(
			"output = %q, expected setup output",
			output.String(),
		)
	}

	if errorOutput.Len() != 0 {
		t.Errorf(
			"error output = %q, expected no output",
			errorOutput.String(),
		)
	}
}

func TestHandleCommandLineSetupFailure(
	t *testing.T,
) {
	originalRunSetupCommand := runSetupCommand

	t.Cleanup(
		func() {
			runSetupCommand =
				originalRunSetupCommand
		},
	)

	runSetupCommand = func(
		io.Writer,
		io.Writer,
	) error {
		return errors.New(
			"unable to write configuration",
		)
	}

	var output bytes.Buffer
	var errorOutput bytes.Buffer

	result := handleCommandLine(
		[]string{"setup"},
		&output,
		&errorOutput,
	)

	if !result.Handled {
		t.Fatal(
			"handleCommandLine() returned Handled=false",
		)
	}

	if result.Err == nil {
		t.Fatal(
			"handleCommandLine() returned nil error",
		)
	}

	if !strings.Contains(
		errorOutput.String(),
		"setup failed",
	) {
		t.Errorf(
			"error output = %q, expected setup failure",
			errorOutput.String(),
		)
	}
}

func TestHandleCommandLineValidate(
	t *testing.T,
) {
	originalLoadConfig := loadValidationConfig
	originalValidateProxmox :=
		validateProxmoxConnection
	originalValidateAdGuard :=
		validateAdGuardConnection

	t.Cleanup(
		func() {
			loadValidationConfig =
				originalLoadConfig
			validateProxmoxConnection =
				originalValidateProxmox
			validateAdGuardConnection =
				originalValidateAdGuard
		},
	)

	loadValidationConfig = func() (
		config.Config,
		error,
	) {
		return config.Config{}, nil
	}

	validateProxmoxConnection = func(
		context.Context,
		config.Config,
	) (string, error) {
		return "9.2.4", nil
	}

	validateAdGuardConnection = func(
		context.Context,
		config.Config,
	) (int, error) {
		return 9, nil
	}

	var output bytes.Buffer
	var errorOutput bytes.Buffer

	result := handleCommandLine(
		[]string{"validate"},
		&output,
		&errorOutput,
	)

	if !result.Handled {
		t.Fatal(
			"handleCommandLine() returned Handled=false",
		)
	}

	if result.Err != nil {
		t.Fatalf(
			"handleCommandLine() error = %v",
			result.Err,
		)
	}

	if !strings.Contains(
		output.String(),
		"Validation completed successfully",
	) {
		t.Errorf(
			"output = %q, expected successful validation",
			output.String(),
		)
	}

	if errorOutput.Len() != 0 {
		t.Errorf(
			"error output = %q, expected no output",
			errorOutput.String(),
		)
	}
}

func TestHandleCommandLineValidateFailure(
	t *testing.T,
) {
	originalLoadConfig := loadValidationConfig

	t.Cleanup(
		func() {
			loadValidationConfig =
				originalLoadConfig
		},
	)

	loadValidationConfig = func() (
		config.Config,
		error,
	) {
		return config.Config{},
			errors.New("missing configuration")
	}

	var output bytes.Buffer
	var errorOutput bytes.Buffer

	result := handleCommandLine(
		[]string{"validate"},
		&output,
		&errorOutput,
	)

	if !result.Handled {
		t.Fatal(
			"handleCommandLine() returned Handled=false",
		)
	}

	if result.Err == nil {
		t.Fatal(
			"handleCommandLine() returned nil error",
		)
	}

	if !strings.Contains(
		errorOutput.String(),
		"validation failed",
	) {
		t.Errorf(
			"error output = %q, expected validation failure",
			errorOutput.String(),
		)
	}
}

func TestHandleCommandLineUnknownCommand(
	t *testing.T,
) {
	var output bytes.Buffer
	var errorOutput bytes.Buffer

	result := handleCommandLine(
		[]string{"unknown"},
		&output,
		&errorOutput,
	)

	if !result.Handled {
		t.Fatal(
			"handleCommandLine() returned Handled=false",
		)
	}

	if result.Err == nil {
		t.Fatal(
			"handleCommandLine() returned nil error",
		)
	}

	if !strings.Contains(
		errorOutput.String(),
		`unknown command "unknown"`,
	) {
		t.Errorf(
			"error output = %q, expected unknown command message",
			errorOutput.String(),
		)
	}

	if !strings.Contains(
		errorOutput.String(),
		"Usage:",
	) {
		t.Errorf(
			"error output = %q, expected usage information",
			errorOutput.String(),
		)
	}
}

func TestHandleCommandLineTooManyArguments(
	t *testing.T,
) {
	var output bytes.Buffer
	var errorOutput bytes.Buffer

	result := handleCommandLine(
		[]string{
			"version",
			"extra",
		},
		&output,
		&errorOutput,
	)

	if !result.Handled {
		t.Fatal(
			"handleCommandLine() returned Handled=false",
		)
	}

	if result.Err == nil {
		t.Fatal(
			"handleCommandLine() returned nil error",
		)
	}

	if !strings.Contains(
		errorOutput.String(),
		"too many arguments",
	) {
		t.Errorf(
			"error output = %q, expected argument error",
			errorOutput.String(),
		)
	}
}
