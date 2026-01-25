// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds all configuration for nats-ls
type Config struct {
	AppMeta struct {
		NameLong         string `mapstructure:"-"`
		NameShort        string `mapstructure:"-"`
		DescriptionShort string `mapstructure:"-"`
		DescriptionLong  string `mapstructure:"-"`
	} `mapstructure:"-"`
	LogLevel                    string `mapstructure:"log_level"`
	NatsURL                     string `mapstructure:"nats_url"`
	NatsPort                    int    `mapstructure:"nats_port"`
	NatsAddress                 string `mapstructure:"nats_address"`
	NatsMaxReconnects           int    `mapstructure:"nats_max_reconnects"`
	NatsReconnectWaitSeconds    int    `mapstructure:"nats_reconnect_wait_seconds"`
	NatsDiscoveryPendingLimit   int    `mapstructure:"nats_discovery_pending_limit"`
	NatsDiscoveryStorageLimitMB int    `mapstructure:"nats_discovery_storage_limit_mb"`
	NatsViewerMessageLimit      int    `mapstructure:"nats_viewer_message_limit"`
	NatsViewerPendingLimit      int    `mapstructure:"nats_viewer_pending_limit"`
	NatsViewerStorageLimitMB    int    `mapstructure:"nats_viewer_storage_limit_mb"`
}

var (
	// appName is the application name used for config directory
	appName = "nats-ls"
	// appDirName is the directory name in home directory
	appDirName = ".nls"
	// configName is the name of the config file (without extension)
	configName = "config"
	// configType is the type/extension of the config file
	configType = "yaml"
)

// Application metadata constants
const (
	AppName        = "nats-ls"
	AppNameShort   = "nls"
	AppDescription = "TUI for NATS"
	AppDescriptionLong = "TUI for inspecting message flow within a NATS server"
)

// GetConfigDir returns the configuration directory path (~/.nls)
func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, appDirName), nil
}

// EnsureConfigDir creates the configuration directory if it doesn't exist
func EnsureConfigDir() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}

	// Create directory with appropriate permissions (0755)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return configDir, nil
}

// GetLogDir returns the log directory path (~/.nls/logs)
func GetLogDir() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "logs"), nil
}

// EnsureLogDir creates the log directory if it doesn't exist
func EnsureLogDir() (string, error) {
	logDir, err := GetLogDir()
	if err != nil {
		return "", err
	}

	// Create directory with appropriate permissions (0755)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return "", err
	}

	return logDir, nil
}

// Load reads the configuration file and returns a Config struct
func Load() (*Config, error) {
	// Create a new viper instance to avoid global state issues
	v := viper.New()

	// Ensure config directory exists and get its path
	configDir, err := EnsureConfigDir()
	if err != nil {
		return nil, err
	}
	v.SetConfigName(configName)
	v.SetConfigType(configType)
	v.AddConfigPath(configDir)

	// Set defaults
	setDefaults(v)

	// Read config file (it's okay if it doesn't exist yet)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file was found but another error occurred
			return nil, err
		}
		// Config file not found, will use defaults
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}

	// If NatsAddress wasn't explicitly provided, construct it from URL and Port
	if cfg.NatsAddress == "" {
		cfg.NatsAddress = fmt.Sprintf("%s:%d", cfg.NatsURL, cfg.NatsPort)
	}

	// Set app metadata from defaults (not user-configurable)
	setMetadata(cfg)

	return cfg, nil
}

// Sets default configuration values
func setDefaults(v *viper.Viper) {
	// Top Level Defaults
	v.SetDefault("log_level", "info")
	v.SetDefault("nats_port", 4222)
	v.SetDefault("nats_url", "127.0.0.1")
	v.SetDefault("nats_max_reconnects", -1) // -1 = infinite reconnects
	v.SetDefault("nats_reconnect_wait_seconds", 2)
	v.SetDefault("nats_discovery_pending_limit", 10000)
	v.SetDefault("nats_discovery_storage_limit_mb", 50)
	v.SetDefault("nats_viewer_message_limit", 100)
	v.SetDefault("nats_viewer_pending_limit", 10000)
	v.SetDefault("nats_viewer_storage_limit_mb", 50)
}

// Sets app Metadata that should not be accessible to the user via the config
func setMetadata(cfg *Config) {
	cfg.AppMeta.NameLong = AppName
	cfg.AppMeta.NameShort = AppNameShort
	cfg.AppMeta.DescriptionShort = AppDescription
	cfg.AppMeta.DescriptionLong = AppDescriptionLong
}

// GenerateDefaultConfigYAML generates a YAML config file with defaults and comments
func GenerateDefaultConfigYAML() (string, error) {
	// Create a viper instance with defaults
	v := viper.New()
	setDefaults(v)

	// Create a map to hold the config with comments
	var buf bytes.Buffer

	buf.WriteString("# nls configuration file\n")
	buf.WriteString("# This file is located at ~/.nls/config.yaml\n\n")

	buf.WriteString("# Logging level (debug, info, warn, error)\n")
	buf.WriteString(fmt.Sprintf("log_level: %s\n\n", v.GetString("log_level")))

	buf.WriteString("# NATS connection settings\n")
	buf.WriteString(fmt.Sprintf("nats_url: %s\n", v.GetString("nats_url")))
	buf.WriteString(fmt.Sprintf("nats_port: %d\n", v.GetInt("nats_port")))
	buf.WriteString("# nats_address: 127.0.0.1:4222  # Alternatively, specify the full address\n\n")

	buf.WriteString("# NATS reconnection settings\n")
	buf.WriteString(fmt.Sprintf("nats_max_reconnects: %d  # -1 = infinite reconnects\n", v.GetInt("nats_max_reconnects")))
	buf.WriteString(fmt.Sprintf("nats_reconnect_wait_seconds: %d\n\n", v.GetInt("nats_reconnect_wait_seconds")))

	buf.WriteString("# NATS discovery settings\n")
	buf.WriteString(fmt.Sprintf("nats_discovery_pending_limit: %d\n", v.GetInt("nats_discovery_pending_limit")))
	buf.WriteString(fmt.Sprintf("nats_discovery_storage_limit_mb: %d\n\n", v.GetInt("nats_discovery_storage_limit_mb")))

	buf.WriteString("# NATS viewer settings\n")
	buf.WriteString(fmt.Sprintf("nats_viewer_message_limit: %d\n", v.GetInt("nats_viewer_message_limit")))
	buf.WriteString(fmt.Sprintf("nats_viewer_pending_limit: %d\n", v.GetInt("nats_viewer_pending_limit")))
	buf.WriteString(fmt.Sprintf("nats_viewer_storage_limit_mb: %d\n", v.GetInt("nats_viewer_storage_limit_mb")))

	return buf.String(), nil
}
