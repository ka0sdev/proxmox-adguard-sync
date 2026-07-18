package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestNewCreatesPrettyTextLogger(t *testing.T) {
	var output bytes.Buffer

	logger, err := New(&output, "info", "text")
	if err != nil {
		t.Fatalf("New() returned an unexpected error: %v", err)
	}

	logger.Info(
		"Application initialized",
		slog.String("component", "test"),
		slog.Int("count", 3),
	)

	logOutput := output.String()

	expectedValues := []string{
		"INFO ",
		"Application initialized",
		"component=test",
		"count=3",
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

func TestPrettyLoggerFormatsStringSlices(
	t *testing.T,
) {
	var output bytes.Buffer

	logger, err := New(&output, "debug", "text")
	if err != nil {
		t.Fatalf("New() returned an unexpected error: %v", err)
	}

	logger.Debug(
		"Selected guest",
		slog.Any(
			"tags",
			[]string{
				"development",
				"testing",
				"uat",
			},
		),
	)

	logOutput := output.String()

	if !strings.Contains(
		logOutput,
		"tags=development,testing,uat",
	) {
		t.Errorf(
			"log output %q does not contain formatted tags",
			logOutput,
		)
	}
}

func TestPrettyLoggerQuotesValuesContainingSpaces(
	t *testing.T,
) {
	var output bytes.Buffer

	logger, err := New(&output, "info", "text")
	if err != nil {
		t.Fatalf("New() returned an unexpected error: %v", err)
	}

	logger.Info(
		"Test message",
		slog.String("value", "contains spaces"),
	)

	logOutput := output.String()

	if !strings.Contains(
		logOutput,
		`value="contains spaces"`,
	) {
		t.Errorf(
			"log output %q does not quote value",
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
		"Application initialized",
		slog.String("component", "test"),
	)

	logOutput := output.String()

	expectedValues := []string{
		`"level":"INFO"`,
		`"msg":"Application initialized"`,
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

func TestLoggerFiltersMessagesBelowConfiguredLevel(
	t *testing.T,
) {
	var output bytes.Buffer

	logger, err := New(&output, "warn", "text")
	if err != nil {
		t.Fatalf("New() returned an unexpected error: %v", err)
	}

	logger.Debug("Debug message")
	logger.Info("Info message")
	logger.Warn("Warning message")

	logOutput := output.String()

	if strings.Contains(logOutput, "Debug message") {
		t.Errorf(
			"log output %q unexpectedly contains debug message",
			logOutput,
		)
	}

	if strings.Contains(logOutput, "Info message") {
		t.Errorf(
			"log output %q unexpectedly contains info message",
			logOutput,
		)
	}

	if !strings.Contains(logOutput, "Warning message") {
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
		t.Fatal(
			"New() returned nil error, expected unsupported-level error",
		)
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
		t.Fatal(
			"New() returned nil error, expected unsupported-format error",
		)
	}

	if !strings.Contains(err.Error(), "unsupported log format") {
		t.Errorf(
			"error = %q, expected unsupported log format error",
			err,
		)
	}
}
