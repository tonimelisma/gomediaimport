package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alexflint/go-arg"
	"gopkg.in/yaml.v2"
)

// args holds the command-line arguments
var args struct {
	SourceDir        string `arg:"positional,required" help:"Source directory for media files"`
	DestDir          string `arg:"--dest" help:"Destination directory for imported media"`
	ConfigFile       string `arg:"--config" help:"Path to config file"`
	OrganizeByDate   bool   `arg:"--organize-by-date" help:"Organize files by date"`
	RenameByDateTime bool   `arg:"--rename-by-date-time" help:"Rename files by date and time"`
	AutoRenameUnique bool   `arg:"--auto-rename-unique" help:"Automatically rename files to ensure uniqueness"`
	Verbose          bool   `arg:"-v,--verbose" help:"Enable verbose output"`
	DryRun           bool   `arg:"--dry-run" help:"Perform a dry run without making changes"`
	SkipThumbnails   bool   `arg:"--skip-thumbnails" help:"Skip thumbnail generation"`
}

// config holds the application configuration
type config struct {
	SourceDir        string `yaml:"source_directory"`
	DestDir          string `yaml:"destination_directory"`
	ConfigFile       string
	OrganizeByDate   bool `yaml:"organize_by_date"`
	RenameByDateTime bool `yaml:"rename_by_date_time"`
	AutoRenameUnique bool `yaml:"auto_rename_unique"`
	Verbose          bool `yaml:"verbose"`
	DryRun           bool `yaml:"dry_run"`
	SkipThumbnails   bool `yaml:"skip_thumbnails"`
}

// setDefaults initializes the config with default values
func setDefaults(cfg *config) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %v", err)
	}

	cfg.DestDir = filepath.Join(homeDir, "Pictures")
	cfg.ConfigFile = filepath.Join(homeDir, ".gomediaimportrc")
	cfg.OrganizeByDate = false
	cfg.RenameByDateTime = false
	cfg.AutoRenameUnique = false
	cfg.Verbose = false
	cfg.DryRun = false
	cfg.SkipThumbnails = false
	return nil
}

// parseConfigFile reads and parses the YAML configuration file
func parseConfigFile(cfg *config) error {
	data, err := os.ReadFile(cfg.ConfigFile)
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

// validateConfig checks if the configuration is valid
func validateConfig(cfg *config) error {
	if cfg.SourceDir == "" {
		return fmt.Errorf("source directory is not specified")
	}

	if cfg.DestDir == "" {
		return fmt.Errorf("destination directory is not specified")
	}

	// Check if source directory exists
	if _, err := os.Stat(cfg.SourceDir); os.IsNotExist(err) {
		return fmt.Errorf("source directory does not exist: %s", cfg.SourceDir)
	}

	return nil
}

func main() {
	// Create an instance of the config struct
	cfg := config{}

	// Set default values first
	if err := setDefaults(&cfg); err != nil {
		fmt.Printf("Error setting defaults: %v\n", err)
		return
	}

	// Parse command-line arguments
	arg.MustParse(&args)

	// Apply config file path from command-line argument if provided
	if args.ConfigFile != "" {
		cfg.ConfigFile = args.ConfigFile
	}

	// Parse configuration file
	if err := parseConfigFile(&cfg); err != nil {
		fmt.Printf("Error parsing config file: %v\n", err)
		return
	}

	// Override with command-line arguments
	if args.SourceDir != "" {
		cfg.SourceDir = args.SourceDir
	}
	if args.DestDir != "" {
		cfg.DestDir = args.DestDir
	}
	if args.OrganizeByDate {
		cfg.OrganizeByDate = args.OrganizeByDate
	}
	if args.RenameByDateTime {
		cfg.RenameByDateTime = args.RenameByDateTime
	}
	if args.AutoRenameUnique {
		cfg.AutoRenameUnique = args.AutoRenameUnique
	}
	if args.Verbose {
		cfg.Verbose = args.Verbose
	}
	if args.DryRun {
		cfg.DryRun = args.DryRun
	}
	if args.SkipThumbnails {
		cfg.SkipThumbnails = args.SkipThumbnails
	}

	// Validate the configuration
	if err := validateConfig(&cfg); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Call the importMedia function
	if err := importMedia(cfg); err != nil {
		fmt.Printf("Error importing media: %v\n", err)
		return
	}
}
