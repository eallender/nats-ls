// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds all configuration for nats-ls
type Config struct {
	// Add your configuration fields here as needed
	// Example:
	// Server struct {
	// 	URL      string
	// 	User     string
	// 	Password string
	// } `mapstructure:"server"`
}

var (
	// AppName is the application name used for config directory
	AppName = "nats-ls"
	// ConfigFileName is the name of the config file (without extension)
	ConfigFileName = "config"
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

	configDir := filepath.Join(configHome, AppName)
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
func Load() (*Config, error) {
	configDir, err := EnsureConfigDir()
	if err != nil {
		return nil, err
	}

	// Configure viper
	viper.SetConfigName(ConfigFileName)
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configDir)

	// Set defaults
	setDefaults()

	// Read config file (it's okay if it doesn't exist yet)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file was found but another error occurred
			return nil, err
		}
		// Config file not found, will use defaults
	}

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save writes the current configuration to disk
func Save(cfg *Config) error {
	configDir, err := EnsureConfigDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(configDir, ConfigFileName+".yaml")
	return viper.WriteConfigAs(configPath)
}

// setDefaults sets default configuration values
func setDefaults() {
	// Add your default values here
	// Example:
	// viper.SetDefault("server.url", "nats://localhost:4222")
}
