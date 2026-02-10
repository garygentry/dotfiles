package logging

import (
	"context"
	"io"
	"log/slog"
	"os"

	"github.com/fatih/color"
)

// Logger provides structured logging interface for the dotfiles system.
type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
	Success(msg string, args ...any)

	With(args ...any) Logger
	WithGroup(name string) Logger
}

// Config holds logger configuration.
type Config struct {
	// Level is the minimum log level (debug, info, warn, error)
	Level string

	// Format is the output format (pretty, json)
	Format string

	// Output is where logs are written (default: os.Stderr)
	Output io.Writer

	// AddSource adds source file/line to log records
	AddSource bool
}

// New creates a new logger with the given configuration.
func New(cfg Config) Logger {
	if cfg.Output == nil {
		cfg.Output = os.Stderr
	}

	level := parseLevel(cfg.Level)

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.AddSource,
	}

	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(cfg.Output, opts)
	} else {
		// Use pretty handler for terminal output
		handler = NewPrettyHandler(cfg.Output, opts)
	}

	return &slogLogger{
		logger: slog.New(handler),
	}
}

// slogLogger wraps slog.Logger to implement our Logger interface.
type slogLogger struct {
	logger *slog.Logger
}

func (l *slogLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

func (l *slogLogger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

func (l *slogLogger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

func (l *slogLogger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

func (l *slogLogger) Success(msg string, args ...any) {
	// Success is logged as info with a success attribute
	args = append(args, "level", "success")
	l.logger.Info(msg, args...)
}

func (l *slogLogger) With(args ...any) Logger {
	return &slogLogger{
		logger: l.logger.With(args...),
	}
}

func (l *slogLogger) WithGroup(name string) Logger {
	return &slogLogger{
		logger: l.logger.WithGroup(name),
	}
}

func parseLevel(levelStr string) slog.Level {
	switch levelStr {
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

// PrettyHandler is a human-readable slog handler for terminal output.
type PrettyHandler struct {
	opts   *slog.HandlerOptions
	output io.Writer
	attrs  []slog.Attr
	groups []string
}

// NewPrettyHandler creates a new pretty handler.
func NewPrettyHandler(output io.Writer, opts *slog.HandlerOptions) *PrettyHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	return &PrettyHandler{
		opts:   opts,
		output: output,
	}
}

func (h *PrettyHandler) Enabled(ctx context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.opts.Level != nil {
		minLevel = h.opts.Level.Level()
	}
	return level >= minLevel
}

func (h *PrettyHandler) Handle(ctx context.Context, r slog.Record) error {
	// Format: [LEVEL] message key=value key=value

	var levelStr string
	switch r.Level {
	case slog.LevelDebug:
		levelStr = color.MagentaString("[DEBUG]")
	case slog.LevelInfo:
		// Check for success level
		isSuccess := false
		r.Attrs(func(a slog.Attr) bool {
			if a.Key == "level" && a.Value.String() == "success" {
				isSuccess = true
				return false
			}
			return true
		})
		if isSuccess {
			levelStr = color.GreenString("[OK]")
		} else {
			levelStr = color.CyanString("[INFO]")
		}
	case slog.LevelWarn:
		levelStr = color.YellowString("[WARN]")
	case slog.LevelError:
		levelStr = color.RedString("[ERROR]")
	default:
		levelStr = "[???]"
	}

	// Build output
	output := levelStr + " " + r.Message

	// Add attributes
	r.Attrs(func(a slog.Attr) bool {
		// Skip internal attributes
		if a.Key == "level" {
			return true
		}
		output += " " + color.HiBlackString(a.Key+"="+a.Value.String())
		return true
	})

	output += "\n"
	_, err := h.output.Write([]byte(output))
	return err
}

func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)

	return &PrettyHandler{
		opts:   h.opts,
		output: h.output,
		attrs:  newAttrs,
		groups: h.groups,
	}
}

func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(h.groups)] = name

	return &PrettyHandler{
		opts:   h.opts,
		output: h.output,
		attrs:  h.attrs,
		groups: newGroups,
	}
}
