package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// config holds the application configuration
type config struct {
	sourceDir  string `yaml:"source_directory"`
	destDir    string `yaml:"destination_directory"`
	configFile string
}

// setDefaults initializes the config with default values
func setDefaults(cfg *config) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %v", err)
	}

	cfg.destDir = filepath.Join(homeDir, "Pictures")
	cfg.configFile = filepath.Join(homeDir, ".gomediaimportrc")
	return nil
}

// parseConfigFile reads and parses the YAML configuration file
func parseConfigFile(cfg *config) error {
	data, err := os.ReadFile(cfg.configFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Config file doesn't exist, just return without an error
			return nil
		}
		return fmt.Errorf("failed to read config file: %v", err)
	}

	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}

	return nil
}

// parseArgs parses command-line arguments and updates the config struct.
// It returns an error if there's an issue with parsing arguments.
func parseArgs(cfg *config) error {
	if len(os.Args) < 2 {
		return fmt.Errorf("source directory is required as the first argument")
	}

	cfg.sourceDir = os.Args[1]
	return nil
}

// validateConfig checks if the configuration is valid
func validateConfig(cfg *config) error {
	if cfg.sourceDir == "" {
		return fmt.Errorf("source directory is not specified")
	}

	if cfg.destDir == "" {
		return fmt.Errorf("destination directory is not specified")
	}

	// Check if source directory exists
	if _, err := os.Stat(cfg.sourceDir); os.IsNotExist(err) {
		return fmt.Errorf("source directory does not exist: %s", cfg.sourceDir)
	}

	return nil
}

func main() {
	// Create an instance of the config struct
	cfg := config{}

	// Set default values
	if err := setDefaults(&cfg); err != nil {
		fmt.Printf("Error setting defaults: %v\n", err)
		return
	}

	// Parse configuration file
	if err := parseConfigFile(&cfg); err != nil {
		fmt.Printf("Error parsing config file: %v\n", err)
		return
	}

	// Parse command-line arguments and update the config
	if err := parseArgs(&cfg); err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("Usage: program <source-directory>")
		return
	}

	// Validate the configuration
	if err := validateConfig(&cfg); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("Source directory:", cfg.sourceDir)
	fmt.Println("Destination directory:", cfg.destDir)
}
