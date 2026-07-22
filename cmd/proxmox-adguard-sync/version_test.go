package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintVersion(t *testing.T) {
	originalVersion := version
	originalCommit := commit
	originalBuildAt := buildAt

	t.Cleanup(func() {
		version = originalVersion
		commit = originalCommit
		buildAt = originalBuildAt
	})

	version = "0.1.0-beta.1"
	commit = "abc1234"
	buildAt = "2026-07-22T18:30:00Z"

	var output bytes.Buffer

	printVersion(&output)

	result := output.String()

	expectedValues := []string{
		"proxmox-adguard-sync 0.1.0-beta.1",
		"commit: abc1234",
		"built: 2026-07-22T18:30:00Z",
	}

	for _, expected := range expectedValues {
		if !strings.Contains(result, expected) {
			t.Errorf(
				"output %q does not contain %q",
				result,
				expected,
			)
		}
	}
}
