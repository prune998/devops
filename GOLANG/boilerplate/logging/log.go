package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

var lvl = new(slog.LevelVar)

func InitDefaultLogger() {
	lvl.Set(slog.LevelInfo)
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: lvl,
	})))
}

func SetGlobalLogLevel(level string) {
	switch strings.ToUpper(level) {
	case "DEBUG":
		lvl.Set(slog.LevelDebug)
	case "INFO":
		lvl.Set(slog.LevelInfo)
	case "WARN":
		lvl.Set(slog.LevelWarn)
	case "ERROR":
		lvl.Set(slog.LevelError)
	default:
		lvl.Set(slog.LevelInfo)
		slog.Warn("invalid log level: will default to INFO", "loglevel", level)
	}
}

func SetGlobalFormat(format string) {
	switch strings.ToUpper(format) {
	case "PLAIN":
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: lvl,
		})))
	case "JSON":
		// already the default
	case "NONE":
		slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: lvl,
		})))
	default:
		slog.Warn("invalid log format, will default to JSON", "format", format)
	}
}
