package handlers

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"

	"github.com/fatih/color"
)

type ColorHandler struct {
	w      io.Writer
	h      slog.Handler
	mu     *sync.Mutex
}

func NewColorHandler(w io.Writer, opts *slog.HandlerOptions) *ColorHandler {
	return &ColorHandler{
		w:  w,
		h:  slog.NewTextHandler(w, opts),
		mu: &sync.Mutex{},
	}
}

func (ch *ColorHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return ch.h.Enabled(ctx, level)
}

func (ch *ColorHandler) Handle(ctx context.Context, r slog.Record) error {
	level := r.Level.String()
	
	switch r.Level {
	case slog.LevelDebug:
		level = color.New(color.FgHiBlack).Sprint(level)
	case slog.LevelInfo:
		level = color.New(color.FgCyan).Sprint(level)
	case slog.LevelWarn:
		level = color.New(color.FgYellow).Sprint(level)
	case slog.LevelError:
		level = color.New(color.FgRed, color.Bold).Sprint(level)
	}

	ch.mu.Lock()
	defer ch.mu.Unlock()

	fmt.Fprintf(ch.w, "[%s] %s %s", 
		r.Time.Format("15:04:05"), 
		level, 
		r.Message,
	)

	r.Attrs(func(a slog.Attr) bool {
		val := fmt.Sprintf("%v", a.Value.Any())
		key := color.New(color.FgHiBlack).Sprint(a.Key)

		switch a.Key {
		case "method":
			val = color.New(color.FgBlue, color.Bold).Sprint(val)
		case "status":
			status, _ := a.Value.Any().(int)
			c := color.FgGreen
			if status >= 400 {
				c = color.FgRed
			} else if status >= 300 {
				c = color.FgYellow
			}
			val = color.New(c, color.Bold).Sprint(val)
		case "path":
			val = color.New(color.FgMagenta).Sprint(val)
		case "duration":
			val = color.New(color.FgYellow).Sprint(val)
		}

		fmt.Fprintf(ch.w, " %s=%s", key, val)
		return true
	})

	fmt.Fprintln(ch.w)
	return nil
}

func (ch *ColorHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ColorHandler{w: ch.w, h: ch.h.WithAttrs(attrs), mu: ch.mu}
}

func (ch *ColorHandler) WithGroup(name string) slog.Handler {
	return &ColorHandler{w: ch.w, h: ch.h.WithGroup(name), mu: ch.mu}
}

type MultiHandler struct {
	handlers []slog.Handler
}

func NewMultiHandler(handlers ...slog.Handler) *MultiHandler {
	return &MultiHandler{handlers: handlers}
}

func (m *MultiHandler) Enabled(ctx context.Context, l slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, l) {
			return true
		}
	}
	return false
}

func (m *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range m.handlers {
		_ = h.Handle(ctx, r)
	}
	return nil
}

func (m *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		newHandlers[i] = h.WithAttrs(attrs)
	}
	return &MultiHandler{handlers: newHandlers}
}

func (m *MultiHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		newHandlers[i] = h.WithGroup(name)
	}
	return &MultiHandler{handlers: newHandlers}
}
