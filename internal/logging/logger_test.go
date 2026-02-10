package logging

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "default config",
			config: Config{
				Level:  "info",
				Format: "pretty",
			},
		},
		{
			name: "json format",
			config: Config{
				Level:  "debug",
				Format: "json",
			},
		},
		{
			name: "with source",
			config: Config{
				Level:     "info",
				Format:    "pretty",
				AddSource: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			tt.config.Output = buf

			logger := New(tt.config)
			if logger == nil {
				t.Fatal("expected non-nil logger")
			}

			// Test basic logging
			logger.Info("test message")

			output := buf.String()
			if len(output) == 0 {
				t.Error("expected log output")
			}
		})
	}
}

func TestLoggerLevels(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Config{
		Level:  "info",
		Format: "pretty",
		Output: buf,
	})

	tests := []struct {
		name     string
		logFunc  func(string, ...any)
		message  string
		wantInOutput bool
	}{
		{
			name:         "info message",
			logFunc:      logger.Info,
			message:      "info test",
			wantInOutput: true,
		},
		{
			name:         "warn message",
			logFunc:      logger.Warn,
			message:      "warn test",
			wantInOutput: true,
		},
		{
			name:         "error message",
			logFunc:      logger.Error,
			message:      "error test",
			wantInOutput: true,
		},
		{
			name:         "success message",
			logFunc:      logger.Success,
			message:      "success test",
			wantInOutput: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc(tt.message)

			output := buf.String()
			contains := strings.Contains(output, tt.message)

			if contains != tt.wantInOutput {
				t.Errorf("message %q in output = %v, want %v\nOutput: %s",
					tt.message, contains, tt.wantInOutput, output)
			}
		})
	}
}

func TestLoggerWith(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Config{
		Level:  "info",
		Format: "pretty",
		Output: buf,
	})

	// Create logger with context
	contextLogger := logger.With("module", "test", "version", "1.0")
	contextLogger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Error("expected message in output")
	}
}

func TestJSONFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Config{
		Level:  "info",
		Format: "json",
		Output: buf,
	})

	logger.Info("json test", "key", "value")

	output := buf.String()
	// JSON output should contain the message
	if !strings.Contains(output, "json test") {
		t.Errorf("expected message in JSON output, got: %s", output)
	}
	// JSON output should be valid JSON (contains {})
	if !strings.Contains(output, "{") || !strings.Contains(output, "}") {
		t.Errorf("expected JSON format, got: %s", output)
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"debug", "DEBUG"},
		{"info", "INFO"},
		{"warn", "WARN"},
		{"warning", "WARN"},
		{"error", "ERROR"},
		{"unknown", "INFO"}, // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			level := parseLevel(tt.input)
			if level.String() != tt.want {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.input, level, tt.want)
			}
		})
	}
}
