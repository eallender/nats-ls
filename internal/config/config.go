// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package config

import (
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
	NatsDiscoveryPendingLimit   int    `mapstructure:"nats_discovery_pending_limit"`
	NatsDiscoveryStorageLimitMB int    `mapstructure:"nats_discovery_storage_limit_mb"`
	NatsViewerPendingLimit      int    `mapstructure:"nats_viewer_pending_limit"`
	NatsViewerStorageLimitMB    int    `mapstructure:"nats_viewer_storage_limit_mb"`
}

var (
	// appName is the application name used for config directory
	appName = "nats-ls"
	// configName is the name of the config file (without extension)
	configName = "config"
	// configType is the type/extension of the config file
	configType = "yaml"
)

// GetConfigDir returns the configuration directory path
// following XDG Base Directory specification
func GetConfigDir() (string, error) {
	// Check for XDG_CONFIG_HOME environment variable
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		// Fall back to ~/.config
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configHome = filepath.Join(homeDir, ".config")
	}

	configDir := filepath.Join(configHome, appName)
	return configDir, nil
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

// Load reads the configuration file and returns a Config struct
func Load(ConfigFile string) (*Config, error) {
	// Create a new viper instance to avoid global state issues
	v := viper.New()

	// If a config file path was provided, verify it exists
	if ConfigFile != "" {
		if _, err := os.Stat(ConfigFile); err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("config file not found: %s", ConfigFile)
			}
			return nil, fmt.Errorf("error accessing config file %s: %w", ConfigFile, err)
		}
		// Use the specific config file provided
		v.SetConfigFile(ConfigFile)
	} else {
		// Use default config directory
		_, err := EnsureConfigDir()
		if err != nil {
			return nil, err
		}
		// Use config directory and name
		configDir, _ := GetConfigDir()
		v.SetConfigName(configName)
		v.SetConfigType(configType)
		v.AddConfigPath(configDir)
	}

	// Set defaults
	setDefaults(v)

	// Read config file (it's okay if it doesn't exist yet)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file was found but another error occurred
			return nil, err
		}
		// Config file not found, will use defaults
	} else {
		// Log which config file was used
		fmt.Fprintf(os.Stderr, "Using config file: %s\n", v.ConfigFileUsed())
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}

	// Set app metadata from defaults (not user-configurable)
	setMetadata(cfg)

	return cfg, nil
}

// Sets default configuration values
func setDefaults(v *viper.Viper) {
	// Top Level Defaults
	v.SetDefault("log_level", "info")
	v.SetDefault("nats_discovery_pending_limit", 10000)
	v.SetDefault("nats_discovery_storage_limit_mb", 50)
	v.SetDefault("nats_viewer_pending_limit", 10000)
	v.SetDefault("nats_viewer_storage_limit_mb", 50)
}

// Sets app Metadata that should not be accessible to the user via the config
func setMetadata(cfg *Config) {
	cfg.AppMeta.NameLong = appName
	cfg.AppMeta.NameShort = "nls"
	cfg.AppMeta.DescriptionShort = "TUI for NATS"
	cfg.AppMeta.DescriptionLong = "TUI for inspecting message flow within a NATS server"
}
