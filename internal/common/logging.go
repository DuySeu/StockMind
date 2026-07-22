package common

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// Package-level logging.
//
// InitLogging installs a slog handler as the process-wide default, so the whole
// codebase logs through the standard library's slog — there is no custom logger
// type to pass around. Logs go to stdout, one line per record:
//
//	17:23:31 | internal/server/chat.handler.go | INFO | server started port=8080
//	 <time>  |      <path from project root>    |LEVEL |    <message>   <attrs...>
//
// Configuration comes from the environment (all optional):
//
//	LOG_LEVEL debug | info | warn | error      (default: info — lower levels are dropped)
//	LOG_COLOR true | false — colored output    (default: true; NO_COLOR=1 also disables)
//
// Usage
//
// Call InitLogging once at startup, right after loading .env:
//
//	common.InitLogging()
//
// Then log from anywhere with the stdlib slog package (no import from this
// package needed). Pass structured data as key/value pairs, not fmt strings:
//
//	slog.Info("server started", "port", 8080)
//	slog.Warn("cache miss, falling back", "key", key)
//	slog.Error("save message", "role", role, "error", err)   // use "error" for errors
//	slog.Debug("bg: persistMessage start", "session", sessionID)
//
// To attach fields to every line from a component, derive a child logger:
//
//	log := slog.With("component", "worker", "doc", docID)
//	log.Info("processing")   // -> ... | INFO | processing component=worker doc=...
//
// Legacy log.Printf calls still work — they are bridged to stdout — but they
// print as raw strings without the time/level/color formatting, so prefer slog.

const (
	ansiReset  = "\x1b[0m"
	ansiGray   = "\x1b[90m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiRed    = "\x1b[31m"
)

// projectRoot is this repo's root (with trailing slash), derived from this
// file's own compile-time path so source lines render relative to the project
// regardless of the working directory.
var projectRoot = func() string {
	_, file, _, _ := runtime.Caller(0)
	return strings.TrimSuffix(filepath.ToSlash(file), "internal/common/logging.go")
}()

var logMu sync.Mutex // serializes writes so lines from different goroutines don't interleave

// InitLogging configures the process-wide logger. Call it once at startup
// (after loading .env). Both slog and the standard log package (log.Printf,
// used across the codebase) are routed to stdout.
func InitLogging() {
	var level slog.Level
	switch strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL"))) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	color := strings.ToLower(os.Getenv("LOG_COLOR")) == "true"
	slog.SetDefault(slog.New(&handler{level: level, color: color}))

	// Bridge the standard log package (raw strings, no pretty format) to
	// stdout; drop its timestamp since these are one-off lines.
	log.SetOutput(os.Stdout)
	log.SetFlags(0)

	slog.Info("logging initialized", "level", level.String())
}

// handler renders slog records as "HH:MM:SS | path | LEVEL | msg attrs...".
type handler struct {
	level slog.Level
	attrs string // pre-rendered " k=v" pairs from WithAttrs
	color bool
}

func (h *handler) Enabled(_ context.Context, l slog.Level) bool { return l >= h.level }

func (h *handler) Handle(_ context.Context, r slog.Record) error {
	attrs := h.attrs
	r.Attrs(func(a slog.Attr) bool {
		attrs += fmt.Sprintf(" %s=%v", a.Key, a.Value.Any())
		return true
	})

	// Render the call site as a path relative to the project root.
	src := "?"
	if r.PC != 0 {
		f, _ := runtime.CallersFrames([]uintptr{r.PC}).Next()
		src = strings.TrimPrefix(filepath.ToSlash(f.File), projectRoot)
	}
	ts, lvl := r.Time.Format("15:04:05"), r.Level.String()

	var line string
	if h.color {
		lvlColor := ansiGray
		switch {
		case r.Level >= slog.LevelError:
			lvlColor = ansiRed
		case r.Level >= slog.LevelWarn:
			lvlColor = ansiYellow
		case r.Level >= slog.LevelInfo:
			lvlColor = ansiGreen
		}
		line = fmt.Sprintf("%s%s%s | %s%s%s | %s%s%s | %s%s\n",
			ansiGray, ts, ansiReset, ansiGray, src, ansiReset,
			lvlColor, lvl, ansiReset, r.Message, attrs)
	} else {
		line = fmt.Sprintf("%s | %s | %s | %s%s\n", ts, src, lvl, r.Message, attrs)
	}

	logMu.Lock()
	defer logMu.Unlock()
	_, err := os.Stdout.WriteString(line)
	return err
}

func (h *handler) WithAttrs(as []slog.Attr) slog.Handler {
	nh := *h
	for _, a := range as {
		nh.attrs += fmt.Sprintf(" %s=%v", a.Key, a.Value.Any())
	}
	return &nh
}

func (h *handler) WithGroup(string) slog.Handler { return h } // groups unused
