package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
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
	testCases := []struct {
		name     string
		argument string
	}{
		{
			name:     "long option",
			argument: "--version",
		},
		{
			name:     "legacy option",
			argument: "-version",
		},
		{
			name:     "command",
			argument: "version",
		},
	}

	for _, testCase := range testCases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				var output bytes.Buffer
				var errorOutput bytes.Buffer

				result := handleCommandLine(
					[]string{
						testCase.argument,
					},
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
	testCases := []struct {
		name     string
		argument string
	}{
		{
			name:     "help command",
			argument: "help",
		},
		{
			name:     "long option",
			argument: "--help",
		},
		{
			name:     "short option",
			argument: "-h",
		},
	}

	for _, testCase := range testCases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				var output bytes.Buffer
				var errorOutput bytes.Buffer

				result := handleCommandLine(
					[]string{
						testCase.argument,
					},
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

				if !strings.Contains(
					output.String(),
					"setup",
				) {
					t.Errorf(
						"output = %q, expected setup command",
						output.String(),
					)
				}

				if !strings.Contains(
					output.String(),
					"validate",
				) {
					t.Errorf(
						"output = %q, expected validate command",
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

func TestHandleCommandLineSetupNotImplemented(
	t *testing.T,
) {
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

	if !errors.Is(
		result.Err,
		errCommandNotImplemented,
	) {
		t.Fatalf(
			"handleCommandLine() error = %v, expected errCommandNotImplemented",
			result.Err,
		)
	}

	if output.Len() != 0 {
		t.Errorf(
			"output = %q, expected no output",
			output.String(),
		)
	}

	if !strings.Contains(
		errorOutput.String(),
		"setup wizard is not implemented yet",
	) {
		t.Errorf(
			"error output = %q, expected setup warning",
			errorOutput.String(),
		)
	}
}

func TestHandleCommandLineValidateNotImplemented(
	t *testing.T,
) {
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

	if !errors.Is(
		result.Err,
		errCommandNotImplemented,
	) {
		t.Fatalf(
			"handleCommandLine() error = %v, expected errCommandNotImplemented",
			result.Err,
		)
	}

	if output.Len() != 0 {
		t.Errorf(
			"output = %q, expected no output",
			output.String(),
		)
	}

	if !strings.Contains(
		errorOutput.String(),
		"configuration validation is not implemented yet",
	) {
		t.Errorf(
			"error output = %q, expected validation warning",
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

	if output.Len() != 0 {
		t.Errorf(
			"output = %q, expected no output",
			output.String(),
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

	if output.Len() != 0 {
		t.Errorf(
			"output = %q, expected no output",
			output.String(),
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
