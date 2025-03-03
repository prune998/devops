package logging

import (
	"log/slog"
	"testing"
)

func TestInitDefaultLogger(t *testing.T) {
	InitDefaultLogger()
	if lvl.Level() != slog.LevelInfo {
		t.Error("calling init default logger should set default level of INFO")
	}
}

func TestSetGlobalLogLevel(t *testing.T) {
	expected := slog.LevelDebug
	SetGlobalLogLevel("DEBUG")
	if lvl.Level() != expected {
		t.Errorf("wrong log level, got '%d', expected '%d'", lvl.Level(), expected)
	}

	expected = slog.LevelInfo
	SetGlobalLogLevel("INFO")
	if lvl.Level() != expected {
		t.Errorf("wrong log level, got '%d', expected '%d'", lvl.Level(), expected)
	}

	expected = slog.LevelWarn
	SetGlobalLogLevel("WARN")
	if lvl.Level() != expected {
		t.Errorf("wrong log level, got '%d', expected '%d'", lvl.Level(), expected)
	}

	expected = slog.LevelError
	SetGlobalLogLevel("ERROR")
	if lvl.Level() != expected {
		t.Errorf("wrong log level, got '%d', expected '%d'", lvl.Level(), expected)
	}
}

func TestSetGlobalLogLevelWithNonExistantLevel(t *testing.T) {
	expected := slog.LevelInfo
	SetGlobalLogLevel("NON_EXISTENT")
	if lvl.Level() != expected {
		t.Errorf("wrong log level, got '%d', expected '%d'", lvl.Level(), expected)
	}
}
