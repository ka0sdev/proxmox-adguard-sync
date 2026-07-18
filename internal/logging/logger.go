package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	FormatText = "text"
	FormatJSON = "json"
)

func New(
	output io.Writer,
	levelName string,
	format string,
) (*slog.Logger, error) {
	level, err := parseLevel(levelName)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(strings.TrimSpace(format)) {
	case FormatText:
		return slog.New(newPrettyHandler(output, level)), nil

	case FormatJSON:
		handler := slog.NewJSONHandler(
			output,
			&slog.HandlerOptions{
				Level: level,
			},
		)

		return slog.New(handler), nil

	default:
		return nil, fmt.Errorf(
			"unsupported log format %q: expected text or json",
			format,
		)
	}
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

type prettyHandler struct {
	output io.Writer
	level  slog.Leveler

	attributes []slog.Attr
	groups     []string

	mutex *sync.Mutex
}

func newPrettyHandler(
	output io.Writer,
	level slog.Leveler,
) *prettyHandler {
	return &prettyHandler{
		output: output,
		level:  level,
		mutex:  &sync.Mutex{},
	}
}

func (h *prettyHandler) Enabled(
	_ context.Context,
	level slog.Level,
) bool {
	return level >= h.level.Level()
}

func (h *prettyHandler) Handle(
	_ context.Context,
	record slog.Record,
) error {
	var builder strings.Builder

	timestamp := record.Time
	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	builder.WriteString(
		timestamp.Local().Format("2006-01-02 15:04:05.000"),
	)
	builder.WriteString("  ")

	builder.WriteString(formatLevel(record.Level))
	builder.WriteString("  ")

	builder.WriteString(record.Message)

	attributes := make([]slog.Attr, 0, len(h.attributes)+record.NumAttrs())
	attributes = append(attributes, h.attributes...)

	record.Attrs(func(attribute slog.Attr) bool {
		attributes = append(attributes, attribute)
		return true
	})

	for _, attribute := range attributes {
		appendAttribute(
			&builder,
			h.groups,
			attribute,
		)
	}

	builder.WriteByte('\n')

	h.mutex.Lock()
	defer h.mutex.Unlock()

	_, err := io.WriteString(h.output, builder.String())

	return err
}

func (h *prettyHandler) WithAttrs(
	attributes []slog.Attr,
) slog.Handler {
	combined := make(
		[]slog.Attr,
		0,
		len(h.attributes)+len(attributes),
	)

	combined = append(combined, h.attributes...)
	combined = append(combined, attributes...)

	return &prettyHandler{
		output:     h.output,
		level:      h.level,
		attributes: combined,
		groups:     append([]string(nil), h.groups...),
		mutex:      h.mutex,
	}
}

func (h *prettyHandler) WithGroup(name string) slog.Handler {
	name = strings.TrimSpace(name)
	if name == "" {
		return h
	}

	groups := append([]string(nil), h.groups...)
	groups = append(groups, name)

	return &prettyHandler{
		output:     h.output,
		level:      h.level,
		attributes: append([]slog.Attr(nil), h.attributes...),
		groups:     groups,
		mutex:      h.mutex,
	}
}

func appendAttribute(
	builder *strings.Builder,
	groups []string,
	attribute slog.Attr,
) {
	attribute.Value = attribute.Value.Resolve()

	if attribute.Equal(slog.Attr{}) {
		return
	}

	if attribute.Value.Kind() == slog.KindGroup {
		nestedGroups := append([]string(nil), groups...)

		if attribute.Key != "" {
			nestedGroups = append(nestedGroups, attribute.Key)
		}

		for _, nestedAttribute := range attribute.Value.Group() {
			appendAttribute(
				builder,
				nestedGroups,
				nestedAttribute,
			)
		}

		return
	}

	key := attribute.Key
	if len(groups) > 0 {
		key = strings.Join(
			append(append([]string(nil), groups...), key),
			".",
		)
	}

	if strings.TrimSpace(key) == "" {
		return
	}

	builder.WriteString("  ")
	builder.WriteString(key)
	builder.WriteByte('=')
	builder.WriteString(formatValue(attribute.Value))
}

func formatLevel(level slog.Level) string {
	switch {
	case level <= slog.LevelDebug:
		return "DEBUG"

	case level < slog.LevelWarn:
		return "INFO "

	case level < slog.LevelError:
		return "WARN "

	default:
		return "ERROR"
	}
}

func formatValue(value slog.Value) string {
	switch value.Kind() {
	case slog.KindString:
		return formatString(value.String())

	case slog.KindBool:
		return strconv.FormatBool(value.Bool())

	case slog.KindInt64:
		return strconv.FormatInt(value.Int64(), 10)

	case slog.KindUint64:
		return strconv.FormatUint(value.Uint64(), 10)

	case slog.KindFloat64:
		return strconv.FormatFloat(
			value.Float64(),
			'f',
			-1,
			64,
		)

	case slog.KindDuration:
		return value.Duration().String()

	case slog.KindTime:
		return value.Time().
			Local().
			Format(time.RFC3339)

	case slog.KindAny:
		return formatAny(value.Any())

	default:
		return formatString(value.String())
	}
}

func formatAny(value any) string {
	switch typedValue := value.(type) {
	case []string:
		return formatString(strings.Join(typedValue, ","))

	case []int:
		values := make([]string, 0, len(typedValue))

		for _, item := range typedValue {
			values = append(values, strconv.Itoa(item))
		}

		return strings.Join(values, ",")

	case error:
		return formatString(typedValue.Error())

	case fmt.Stringer:
		return formatString(typedValue.String())

	case nil:
		return "null"
	}

	reflected := reflect.ValueOf(value)

	if reflected.IsValid() &&
		(reflected.Kind() == reflect.Slice ||
			reflected.Kind() == reflect.Array) {
		values := make([]string, 0, reflected.Len())

		for index := 0; index < reflected.Len(); index++ {
			values = append(
				values,
				fmt.Sprint(reflected.Index(index).Interface()),
			)
		}

		return formatString(strings.Join(values, ","))
	}

	return formatString(fmt.Sprint(value))
}

func formatString(value string) string {
	if value == "" {
		return `""`
	}

	requiresQuotes := strings.IndexFunc(
		value,
		func(character rune) bool {
			return unicode.IsSpace(character) ||
				character == '=' ||
				character == '"'
		},
	) >= 0

	if requiresQuotes {
		return strconv.Quote(value)
	}

	return value
}
