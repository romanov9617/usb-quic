// Package logging contains application logging helpers.
package logging

import (
	"context"
	"fmt"
	"log/slog"
)

// MultiHandler fans slog records out to several handlers.
type MultiHandler struct {
	handlers []slog.Handler
}

// NewMultiHandler creates a handler that writes each record to all enabled handlers.
func NewMultiHandler(handlers ...slog.Handler) *MultiHandler {
	filteredHandlers := make([]slog.Handler, 0, len(handlers))
	for _, handler := range handlers {
		if handler != nil {
			filteredHandlers = append(filteredHandlers, handler)
		}
	}

	return &MultiHandler{
		handlers: filteredHandlers,
	}
}

// Enabled reports whether at least one child handler accepts level.
func (handler *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, child := range handler.handlers {
		if child.Enabled(ctx, level) {
			return true
		}
	}

	return false
}

// Handle sends record to each enabled child handler.
func (handler *MultiHandler) Handle(ctx context.Context, record slog.Record) error {
	for index, child := range handler.handlers {
		if !child.Enabled(ctx, record.Level) {
			continue
		}

		childRecord := record.Clone()

		err := child.Handle(ctx, childRecord)
		if err != nil {
			return fmt.Errorf("handle log record with handler %d: %w", index, err)
		}
	}

	return nil
}

// WithAttrs returns a grouped handler with attrs attached to every child handler.
func (handler *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	children := make([]slog.Handler, 0, len(handler.handlers))
	for _, child := range handler.handlers {
		children = append(children, child.WithAttrs(attrs))
	}

	return NewMultiHandler(children...)
}

// WithGroup returns a grouped handler with group attached to every child handler.
func (handler *MultiHandler) WithGroup(name string) slog.Handler {
	children := make([]slog.Handler, 0, len(handler.handlers))
	for _, child := range handler.handlers {
		children = append(children, child.WithGroup(name))
	}

	return NewMultiHandler(children...)
}
