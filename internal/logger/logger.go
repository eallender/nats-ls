// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package logger

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/eallender/nats-ls/internal/config"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Log *slog.Logger

// Init initializes the global logger with automatic rotation
func Init(logLevel string) error {
	level := GetLevel(logLevel)

	logDir, err := config.EnsureConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get log directory: %w", err)
	}

	logFile := filepath.Join(logDir, "nls.log")

	// Clear existing log file on startup
	if err := os.Truncate(logFile, 0); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to truncate log file: %w", err)
	}

	// Create rotating file logger with size limits
	fileWriter := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    10,    // megabytes - rotate when file reaches this size
		MaxBackups: 0,     // don't keep any old backups
		MaxAge:     0,     // don't delete based on age
		Compress:   false, // don't compress old logs
	}

	handler := slog.NewTextHandler(fileWriter, &slog.HandlerOptions{Level: level})
	Log = slog.New(handler)
	slog.SetDefault(Log)

	// Log where the log file is located
	Log.Info("Logger initialized", "log_file", logFile, "level", logLevel, "max_size_mb", 10)

	return nil
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
