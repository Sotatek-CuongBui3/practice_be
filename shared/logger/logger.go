package logger

import (
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

// Config holds logger configuration
type Config struct {
	Level        string // debug, info, warn, error
	Format       string // json, console
	Output       string // stdout, stderr, or file path
	EnableSource bool   // Enable source code location
	TimeFormat   string // Time format for console output
}

// Logger wraps slog.Logger
type Logger struct {
	*slog.Logger
}

// New creates a new logger instance
func New(config *Config) (*Logger, error) {
	level := parseLevel(config.Level)

	var writer io.Writer
	switch config.Output {
	case "stderr":
		writer = os.Stderr
	case "stdout", "":
		writer = os.Stdout
	default:
		// TODO: Support file output
		writer = os.Stdout
	}

	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: config.EnableSource,
	}

	switch config.Format {
	case "json":
		handler = slog.NewJSONHandler(writer, opts)
	case "console", "":
		// Use tint for colorful console output
		timeFormat := config.TimeFormat
		if timeFormat == "" {
			timeFormat = time.RFC3339
		}

		handler = tint.NewHandler(writer, &tint.Options{
			Level:      level,
			AddSource:  config.EnableSource,
			TimeFormat: timeFormat,
			NoColor:    false, // Enable colors
		})
	default:
		handler = slog.NewJSONHandler(writer, opts)
	}

	logger := slog.New(handler)

	return &Logger{Logger: logger}, nil
}

// NewDefault creates a logger with default settings (console format, info level)
func NewDefault() *Logger {
	handler := tint.NewHandler(os.Stdout, &tint.Options{
		Level:      slog.LevelInfo,
		TimeFormat: time.TimeOnly,
		NoColor:    false,
	})

	return &Logger{Logger: slog.New(handler)}
}

// parseLevel converts string level to slog.Level
func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithGroup creates a new logger with a group namespace
func (l *Logger) WithGroup(name string) *Logger {
	return &Logger{Logger: l.Logger.WithGroup(name)}
}

// WithAttrs creates a new logger with additional attributes
func (l *Logger) WithAttrs(attrs ...slog.Attr) *Logger {
	return &Logger{Logger: l.Logger.With(attrsToAny(attrs)...)}
}

// With creates a new logger with additional key-value pairs
func (l *Logger) With(args ...any) *Logger {
	return &Logger{Logger: l.Logger.With(args...)}
}

// attrsToAny converts []slog.Attr to []any
func attrsToAny(attrs []slog.Attr) []any {
	result := make([]any, len(attrs))
	for i, attr := range attrs {
		result[i] = attr
	}
	return result
}
