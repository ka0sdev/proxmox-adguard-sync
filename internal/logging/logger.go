package logging

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

const (
	FormatText = "text"
	FormatJSON = "json"
)

func New(output io.Writer, levelName, format string) (*slog.Logger, error) {
	level, err := parseLevel(levelName)
	if err != nil {
		return nil, err
	}

	options := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler

	switch strings.ToLower(strings.TrimSpace(format)) {
	case FormatText:
		handler = slog.NewTextHandler(output, options)
	case FormatJSON:
		handler = slog.NewJSONHandler(output, options)
	default:
		return nil, fmt.Errorf(
			"unsupported log format %q: expected text or json",
			format,
		)
	}

	return slog.New(handler), nil
}

func parseLevel(value string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf(
			"unsupported log level %q: expected debug, info, warn, or error",
			value,
		)
	}
}
