package jiraworklog

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"
)

type LoggerOptions struct {
	Application string
	LogFile     string
	Level       string
}

// NewLogger creates a new structured logger using slog with best practices
func NewLogger(options LoggerOptions) *slog.Logger {
	if options.Level == "" {
		options.Level = "warn"
	}

	// Parse log level
	level := parseLevel(options.Level)

	// Create handler options
	handlerOpts := &slog.HandlerOptions{
		Level: level,
		// Add source location for errors and warnings
		AddSource: level <= slog.LevelInfo,
		// Replace default attributes for better readability
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Rename "time" to "timestamp" and format as RFC3339
			if a.Key == slog.TimeKey {
				return slog.String("timestamp", a.Value.Time().Format(time.RFC3339))
			}
			// Rename "msg" to "message"
			if a.Key == slog.MessageKey {
				return slog.String("message", a.Value.String())
			}
			// Keep level as "level" but format as uppercase
			if a.Key == slog.LevelKey {
				return slog.String("level", a.Value.String())
			}
			return a
		},
	}

	var writer io.Writer = os.Stdout

	// Setup log file if specified
	if options.LogFile != "" {
		file, err := os.OpenFile(options.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			// Write to both file and stdout
			writer = io.MultiWriter(file, os.Stdout)
		} else {
			// Fallback to stdout with warning
			slog.Warn("Failed to open log file, using stdout only", "error", err, "file", options.LogFile)
		}
	}

	// Create handler - using JSON for production-ready structured logging
	// For human-readable logs, use slog.NewTextHandler instead
	handler := slog.NewTextHandler(writer, handlerOpts)

	// Create logger with application context
	logger := slog.New(handler).With(
		slog.String("app", options.Application),
	)

	// Set as default logger
	slog.SetDefault(logger)

	return logger
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
		return slog.LevelWarn
	}
}

// Helper functions for common logging patterns

// LogError logs an error with additional context
func LogError(logger *slog.Logger, msg string, err error, attrs ...any) {
	args := append([]any{"error", err}, attrs...)
	logger.Error(msg, args...)
}

// LogErrorContext logs an error with context
func LogErrorContext(ctx context.Context, logger *slog.Logger, msg string, err error, attrs ...any) {
	args := append([]any{"error", err}, attrs...)
	logger.ErrorContext(ctx, msg, args...)
}

// LogInfo logs info with structured attributes
func LogInfo(logger *slog.Logger, msg string, attrs ...any) {
	logger.Info(msg, attrs...)
}

// LogWarn logs a warning with structured attributes
func LogWarn(logger *slog.Logger, msg string, attrs ...any) {
	logger.Warn(msg, attrs...)
}

// LogDebug logs debug information
func LogDebug(logger *slog.Logger, msg string, attrs ...any) {
	logger.Debug(msg, attrs...)
}

// WithAttrs creates a new logger with additional attributes
func WithAttrs(logger *slog.Logger, attrs ...any) *slog.Logger {
	return logger.With(attrs...)
}

// WithGroup creates a new logger with a group
func WithGroup(logger *slog.Logger, name string) *slog.Logger {
	return logger.WithGroup(name)
}
