// Package logging configures application loggers.
package logging

import (
	"io"
	"log/slog"

	"usb-quic/internal/adapter/logging/multi"
)

// DefaultLevel is the default application logging level.
const DefaultLevel = slog.LevelDebug

// Logger is the application structured logger.
type Logger = slog.Logger

// LevelVar is a mutable slog level.
type LevelVar = slog.LevelVar

// NewLevel creates a mutable logging level.
func NewLevel(level slog.Level) *LevelVar {
	levelVar := &slog.LevelVar{}
	levelVar.Set(level)

	return levelVar
}

// NewDefaultLevel creates a mutable debug logging level.
func NewDefaultLevel() *LevelVar {
	return NewLevel(DefaultLevel)
}

// NewVerboseLevel creates a mutable level for Linux-style -v verbosity.
func NewVerboseLevel(verbose bool) *LevelVar {
	level := NewDefaultLevel()
	SetVerboseLevel(level, verbose)

	return level
}

// SetVerboseLevel applies Linux-style verbosity: info by default, debug with -v.
func SetVerboseLevel(level *LevelVar, verbose bool) {
	if level == nil {
		return
	}

	if verbose {
		level.Set(slog.LevelDebug)

		return
	}

	level.Set(slog.LevelInfo)
}

// NewTextHandler creates the default text slog handler.
func NewTextHandler(writer io.Writer, level slog.Leveler) slog.Handler {
	return slog.NewTextHandler(writer, &slog.HandlerOptions{
		AddSource:   false,
		Level:       level,
		ReplaceAttr: nil,
	})
}

// NewTextLogger creates a logger backed by a text handler.
func NewTextLogger(writer io.Writer, level slog.Leveler) *slog.Logger {
	return slog.New(multi.NewMultiHandler(NewTextHandler(writer, level)))
}

// NewDefaultLogger creates the default text logger with debug level.
func NewDefaultLogger(writer io.Writer) *slog.Logger {
	return NewTextLogger(writer, NewDefaultLevel())
}
