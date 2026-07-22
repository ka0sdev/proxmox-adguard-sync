package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestHandleCommandLineVersion(t *testing.T) {
	var output bytes.Buffer

	handled := handleCommandLine(
		[]string{"--version"},
		&output,
	)

	if !handled {
		t.Fatal(
			"handleCommandLine() returned false",
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
}

func TestHandleCommandLineVersionAlias(t *testing.T) {
	var output bytes.Buffer

	handled := handleCommandLine(
		[]string{"-version"},
		&output,
	)

	if !handled {
		t.Fatal(
			"handleCommandLine() returned false",
		)
	}
}

func TestHandleCommandLineWithoutArguments(
	t *testing.T,
) {
	var output bytes.Buffer

	handled := handleCommandLine(nil, &output)

	if handled {
		t.Fatal(
			"handleCommandLine() returned true",
		)
	}

	if output.Len() != 0 {
		t.Errorf(
			"output = %q, expected no output",
			output.String(),
		)
	}
}

func TestHandleCommandLineUnknownArgument(
	t *testing.T,
) {
	var output bytes.Buffer

	handled := handleCommandLine(
		[]string{"--unknown"},
		&output,
	)

	if handled {
		t.Fatal(
			"handleCommandLine() returned true",
		)
	}
}
