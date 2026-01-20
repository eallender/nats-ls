// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/eallender/nats-ls/internal/config"
	"github.com/eallender/nats-ls/internal/logger"
	"github.com/eallender/nats-ls/internal/tui"
	"github.com/spf13/cobra"
)

var (
	// The app configuration
	cfg *config.Config
	// The config file path
	configFile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "",
	Short: "",
	Long:  "",

	Run: func(cmd *cobra.Command, args []string) {
		if err := tui.Run(); err != nil {
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
	// Set up flag parsing to happen before command execution
	cobra.OnInitialize(initConfig)

	// CLI Flags
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Config file path. Default: ~/.config/nats-ls/config.yaml")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Load configuration
	var err error
	cfg, err = config.Load(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger.Init(cfg.LogLevel)

	// Log the loaded configuration
	configJSON, _ := json.MarshalIndent(cfg, "", "  ")
	logger.Log.Debug("Configuration loaded", "config", string(configJSON))

	// Update root command from config
	rootCmd.Use = cfg.AppMeta.NameShort
	rootCmd.Short = cfg.AppMeta.DescriptionShort
	rootCmd.Long = cfg.AppMeta.DescriptionLong
}
