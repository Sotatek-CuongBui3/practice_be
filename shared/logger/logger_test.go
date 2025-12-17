package logger

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantErr   bool
		checkFunc func(t *testing.T, logger *Logger, output *bytes.Buffer)
	}{
		{
			name: "json format with debug level",
			config: &Config{
				Level:        "debug",
				Format:       "json",
				Output:       "stdout",
				EnableSource: false,
				TimeFormat:   time.RFC3339,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, logger *Logger, output *bytes.Buffer) {
				logger.Debug("test debug message", slog.String("key", "value"))

				var logEntry map[string]interface{}
				err := json.Unmarshal(output.Bytes(), &logEntry)
				require.NoError(t, err)

				assert.Equal(t, "DEBUG", logEntry["level"])
				assert.Equal(t, "test debug message", logEntry["msg"])
				assert.Equal(t, "value", logEntry["key"])
				assert.Contains(t, logEntry, "time")
			},
		},
		{
			name: "json format with info level",
			config: &Config{
				Level:        "info",
				Format:       "json",
				Output:       "stdout",
				EnableSource: false,
				TimeFormat:   time.RFC3339,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, logger *Logger, output *bytes.Buffer) {
				logger.Debug("debug message")
				logger.Info("info message", slog.String("type", "test"))

				lines := strings.Split(strings.TrimSpace(output.String()), "\n")
				// Debug should not be logged
				assert.Len(t, lines, 1)

				var logEntry map[string]interface{}
				err := json.Unmarshal([]byte(lines[0]), &logEntry)
				require.NoError(t, err)

				assert.Equal(t, "INFO", logEntry["level"])
				assert.Equal(t, "info message", logEntry["msg"])
				assert.Equal(t, "test", logEntry["type"])
			},
		},
		{
			name: "json format with warn level",
			config: &Config{
				Level:        "warn",
				Format:       "json",
				Output:       "stdout",
				EnableSource: false,
				TimeFormat:   time.RFC3339,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, logger *Logger, output *bytes.Buffer) {
				logger.Info("info message")
				logger.Warn("warn message", slog.String("severity", "high"))

				lines := strings.Split(strings.TrimSpace(output.String()), "\n")
				// Info should not be logged
				assert.Len(t, lines, 1)

				var logEntry map[string]interface{}
				err := json.Unmarshal([]byte(lines[0]), &logEntry)
				require.NoError(t, err)

				assert.Equal(t, "WARN", logEntry["level"])
				assert.Equal(t, "warn message", logEntry["msg"])
				assert.Equal(t, "high", logEntry["severity"])
			},
		},
		{
			name: "json format with error level",
			config: &Config{
				Level:        "error",
				Format:       "json",
				Output:       "stdout",
				EnableSource: false,
				TimeFormat:   time.RFC3339,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, logger *Logger, output *bytes.Buffer) {
				logger.Warn("warn message")
				logger.Error("error message", slog.String("code", "500"))

				lines := strings.Split(strings.TrimSpace(output.String()), "\n")
				// Warn should not be logged
				assert.Len(t, lines, 1)

				var logEntry map[string]interface{}
				err := json.Unmarshal([]byte(lines[0]), &logEntry)
				require.NoError(t, err)

				assert.Equal(t, "ERROR", logEntry["level"])
				assert.Equal(t, "error message", logEntry["msg"])
				assert.Equal(t, "500", logEntry["code"])
			},
		},
		{
			name: "console format with colors",
			config: &Config{
				Level:        "info",
				Format:       "console",
				Output:       "stdout",
				EnableSource: false,
				TimeFormat:   time.RFC3339,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, logger *Logger, output *bytes.Buffer) {
				logger.Info("console test")

				// Console format should contain the message (tint uses "INF" not "INFO")
				logOutput := output.String()
				assert.Contains(t, logOutput, "INF")
				assert.Contains(t, logOutput, "console test")
			},
		},
		{
			name: "with source location enabled",
			config: &Config{
				Level:        "info",
				Format:       "json",
				Output:       "stdout",
				EnableSource: true,
				TimeFormat:   time.RFC3339,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, logger *Logger, output *bytes.Buffer) {
				logger.Info("message with source")

				var logEntry map[string]interface{}
				err := json.Unmarshal(output.Bytes(), &logEntry)
				require.NoError(t, err)

				assert.Contains(t, logEntry, "source")
				source := logEntry["source"].(map[string]interface{})
				assert.Contains(t, source, "function")
				assert.Contains(t, source, "file")
				assert.Contains(t, source, "line")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output
			output := &bytes.Buffer{}

			// Override output for testing
			originalConfig := *tt.config
			originalConfig.writer = output

			logger, err := New(&originalConfig)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, logger)
			} else {
				require.NoError(t, err)
				require.NotNil(t, logger)

				if tt.checkFunc != nil {
					tt.checkFunc(t, logger, output)
				}
			}
		})
	}
}

func TestNewDefault(t *testing.T) {
	logger := NewDefault()
	require.NotNil(t, logger)
	assert.NotNil(t, logger.Logger)
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected slog.Level
	}{
		{
			name:     "debug level",
			level:    "debug",
			expected: slog.LevelDebug,
		},
		{
			name:     "info level",
			level:    "info",
			expected: slog.LevelInfo,
		},
		{
			name:     "warn level",
			level:    "warn",
			expected: slog.LevelWarn,
		},
		{
			name:     "error level",
			level:    "error",
			expected: slog.LevelError,
		},
		{
			name:     "uppercase debug",
			level:    "DEBUG",
			expected: slog.LevelInfo, // parseLevel is case-sensitive, defaults to info
		},
		{
			name:     "uppercase info",
			level:    "INFO",
			expected: slog.LevelInfo, // parseLevel is case-sensitive, but "INFO" also defaults to info
		},
		{
			name:     "invalid level defaults to info",
			level:    "invalid",
			expected: slog.LevelInfo,
		},
		{
			name:     "empty string defaults to info",
			level:    "",
			expected: slog.LevelInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLevel(tt.level)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLogger_WithGroup(t *testing.T) {
	output := &bytes.Buffer{}

	logger, err := New(&Config{
		Level:        "info",
		Format:       "json",
		Output:       "stdout",
		EnableSource: false,
		TimeFormat:   time.RFC3339,
		writer:       output,
	})
	require.NoError(t, err)

	groupLogger := logger.WithGroup("mygroup")
	require.NotNil(t, groupLogger)

	groupLogger.Info("test message", slog.String("key", "value"))

	var logEntry map[string]interface{}
	err = json.Unmarshal(output.Bytes(), &logEntry)
	require.NoError(t, err)

	// Check that the group exists
	assert.Contains(t, logEntry, "mygroup")
	group := logEntry["mygroup"].(map[string]interface{})
	assert.Equal(t, "value", group["key"])
}

func TestLogger_WithAttrs(t *testing.T) {
	output := &bytes.Buffer{}

	logger, err := New(&Config{
		Level:        "info",
		Format:       "json",
		Output:       "stdout",
		EnableSource: false,
		TimeFormat:   time.RFC3339,
		writer:       output,
	})
	require.NoError(t, err)

	attrLogger := logger.WithAttrs(
		slog.String("request_id", "12345"),
		slog.String("user_id", "user-67890"),
	)
	require.NotNil(t, attrLogger)

	attrLogger.Info("test message")

	var logEntry map[string]interface{}
	err = json.Unmarshal(output.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "12345", logEntry["request_id"])
	assert.Equal(t, "user-67890", logEntry["user_id"])
	assert.Equal(t, "test message", logEntry["msg"])
}

func TestLogger_With(t *testing.T) {
	output := &bytes.Buffer{}

	logger, err := New(&Config{
		Level:        "info",
		Format:       "json",
		Output:       "stdout",
		EnableSource: false,
		TimeFormat:   time.RFC3339,
		writer:       output,
	})
	require.NoError(t, err)

	contextLogger := logger.With(
		slog.String("service", "api"),
		slog.Int("version", 1),
	)
	require.NotNil(t, contextLogger)

	contextLogger.Info("operation complete")

	var logEntry map[string]interface{}
	err = json.Unmarshal(output.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "api", logEntry["service"])
	assert.Equal(t, float64(1), logEntry["version"]) // JSON numbers are float64
	assert.Equal(t, "operation complete", logEntry["msg"])
}

func TestLogger_MultipleAttributes(t *testing.T) {
	output := &bytes.Buffer{}

	logger, err := New(&Config{
		Level:        "info",
		Format:       "json",
		Output:       "stdout",
		EnableSource: false,
		TimeFormat:   time.RFC3339,
		writer:       output,
	})
	require.NoError(t, err)

	logger.Info("complex log",
		slog.String("string_val", "test"),
		slog.Int("int_val", 42),
		slog.Bool("bool_val", true),
		slog.Float64("float_val", 3.14),
	)

	var logEntry map[string]interface{}
	err = json.Unmarshal(output.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "test", logEntry["string_val"])
	assert.Equal(t, float64(42), logEntry["int_val"])
	assert.Equal(t, true, logEntry["bool_val"])
	assert.Equal(t, 3.14, logEntry["float_val"])
}
