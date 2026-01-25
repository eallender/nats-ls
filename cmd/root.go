// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/eallender/nats-ls/internal/config"
	"github.com/eallender/nats-ls/internal/logger"
	"github.com/eallender/nats-ls/internal/tui"
	"github.com/spf13/cobra"
)

var (
	// The app configuration
	cfg *config.Config
	// Flag to generate default config
	createConfig bool
	// NATS connection override flags
	natsServer string
	natsURL    string
	natsPort   int
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   config.AppNameShort,
	Short: config.AppDescription,
	Long:  config.AppDescriptionLong,

	Run: func(cmd *cobra.Command, args []string) {
		// If --generate-config flag is set, generate config and exit
		if createConfig {
			if err := generateDefaultConfig(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		}

		// Load configuration
		if err := loadConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Run the TUI
		if err := tui.Run(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// CLI Flags
	rootCmd.Flags().BoolVar(&createConfig, "generate-config", false, "Generate default config file at ~/.nats-ls/config.yaml and exit")

	// NATS connection flags (override config file)
	rootCmd.Flags().StringVar(&natsServer, "server", "", "NATS server address (overrides config, e.g., 127.0.0.1:4222)")
	rootCmd.Flags().StringVar(&natsURL, "url", "", "NATS server URL (overrides config, e.g., 127.0.0.1)")
	rootCmd.Flags().IntVar(&natsPort, "port", 0, "NATS server port (overrides config, e.g., 4222)")

	// Make --server mutually exclusive with --url and --port
	rootCmd.MarkFlagsMutuallyExclusive("server", "url")
	rootCmd.MarkFlagsMutuallyExclusive("server", "port")
}

// loadConfig reads in config file and initializes the application
func loadConfig() error {
	var err error
	cfg, err = config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Apply CLI flag overrides
	if natsServer != "" {
		cfg.NatsAddress = natsServer
	}
	if natsURL != "" {
		cfg.NatsURL = natsURL
	}
	if natsPort != 0 {
		cfg.NatsPort = natsPort
	}

	// Reconstruct NatsAddress if URL or Port were provided
	if (natsURL != "" || natsPort != 0) && natsServer == "" {
		cfg.NatsAddress = fmt.Sprintf("%s:%d", cfg.NatsURL, cfg.NatsPort)
	}

	// Initialize logger
	logger.Init(cfg.LogLevel)

	// Log the loaded configuration
	configJSON, _ := json.MarshalIndent(cfg, "", "  ")
	logger.Log.Debug("Configuration loaded", "config", string(configJSON))

	return nil
}

func generateDefaultConfig() error {
	// Ensure the config directory exists
	configDir, err := config.EnsureConfigDir()
	if err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Config file already exists at: %s\n", configPath)
		fmt.Print("Overwrite? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	// Generate config content from defaults using the config package
	configContent, err := config.GenerateDefaultConfigYAML()
	if err != nil {
		return fmt.Errorf("failed to generate config content: %w", err)
	}

	// Write the config file
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Configuration file created at: %s\n", configPath)
	return nil
}
