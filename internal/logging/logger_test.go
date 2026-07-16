package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestNewCreatesTextLogger(t *testing.T) {
	var output bytes.Buffer

	logger, err := New(&output, "info", "text")
	if err != nil {
		t.Fatalf("New() returned an unexpected error: %v", err)
	}

	logger.Info(
		"application initialized",
		slog.String("component", "test"),
	)

	logOutput := output.String()

	if !strings.Contains(logOutput, "level=INFO") {
		t.Errorf(
			"log output %q does not contain INFO level",
			logOutput,
		)
	}

	if !strings.Contains(logOutput, `msg="application initialized"`) {
		t.Errorf(
			"log output %q does not contain expected message",
			logOutput,
		)
	}

	if !strings.Contains(logOutput, "component=test") {
		t.Errorf(
			"log output %q does not contain expected attribute",
			logOutput,
		)
	}
}

func TestNewCreatesJSONLogger(t *testing.T) {
	var output bytes.Buffer

	logger, err := New(&output, "info", "json")
	if err != nil {
		t.Fatalf("New() returned an unexpected error: %v", err)
	}

	logger.Info(
		"application initialized",
		slog.String("component", "test"),
	)

	logOutput := output.String()

	expectedValues := []string{
		`"level":"INFO"`,
		`"msg":"application initialized"`,
		`"component":"test"`,
	}

	for _, expectedValue := range expectedValues {
		if !strings.Contains(logOutput, expectedValue) {
			t.Errorf(
				"log output %q does not contain %q",
				logOutput,
				expectedValue,
			)
		}
	}
}

func TestLoggerFiltersMessagesBelowConfiguredLevel(t *testing.T) {
	var output bytes.Buffer

	logger, err := New(&output, "warn", "text")
	if err != nil {
		t.Fatalf("New() returned an unexpected error: %v", err)
	}

	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warning message")

	logOutput := output.String()

	if strings.Contains(logOutput, "debug message") {
		t.Errorf(
			"log output %q unexpectedly contains debug message",
			logOutput,
		)
	}

	if strings.Contains(logOutput, "info message") {
		t.Errorf(
			"log output %q unexpectedly contains info message",
			logOutput,
		)
	}

	if !strings.Contains(logOutput, "warning message") {
		t.Errorf(
			"log output %q does not contain warning message",
			logOutput,
		)
	}
}

func TestNewRejectsUnsupportedLevel(t *testing.T) {
	var output bytes.Buffer

	_, err := New(&output, "verbose", "text")
	if err == nil {
		t.Fatal("New() returned nil error, expected unsupported-level error")
	}

	if !strings.Contains(err.Error(), "unsupported log level") {
		t.Errorf(
			"error = %q, expected unsupported log level error",
			err,
		)
	}
}

func TestNewRejectsUnsupportedFormat(t *testing.T) {
	var output bytes.Buffer

	_, err := New(&output, "info", "xml")
	if err == nil {
		t.Fatal("New() returned nil error, expected unsupported-format error")
	}

	if !strings.Contains(err.Error(), "unsupported log format") {
		t.Errorf(
			"error = %q, expected unsupported log format error",
			err,
		)
	}
}
