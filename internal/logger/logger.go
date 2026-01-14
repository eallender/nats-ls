// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package logger

import (
	"log/slog"
	"os"
)

var Log *slog.Logger

// Init initializes the global logger
func Init() {
	// Create a new text handler that writes to stderr
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	Log = slog.New(handler)
	slog.SetDefault(Log)
}

// SetLevel sets the log level
func SetLevel(level slog.Level) {
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})
	Log = slog.New(handler)
	slog.SetDefault(Log)
}
