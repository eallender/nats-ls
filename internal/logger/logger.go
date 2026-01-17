// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package logger

import (
	"log/slog"
	"os"
	"strings"
)

var Log *slog.Logger

// Init initializes the global logger
func Init(logLevel string) {
	SetLevel(GetLevel(logLevel))
}

// SetLevel sets the log level
func SetLevel(level slog.Level) {
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})
	Log = slog.New(handler)
	slog.SetDefault(Log)
}

// Gets the log level from the given string
func GetLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	}

	return slog.LevelInfo
}
