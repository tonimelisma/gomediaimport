package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexflint/go-arg"
	"gopkg.in/yaml.v3"
)

// args holds the command-line arguments
var args struct {
	SourceDir          string `arg:"positional,required" help:"Source directory for media files"`
	DestDir            string `arg:"--dest" help:"Destination directory for imported media"`
	ConfigFile         string `arg:"--config" help:"Path to config file"`
	OrganizeByDate     bool   `arg:"--organize-by-date" help:"Organize files by date"`
	RenameByDateTime   bool   `arg:"--rename-by-date-time" help:"Rename files by date and time"`
	ChecksumDuplicates bool   `arg:"--checksum-duplicates" help:"Use checksums to identify duplicates"`
	Verbose            bool   `arg:"-v,--verbose" help:"Enable verbose output"`
	DryRun             bool   `arg:"--dry-run" help:"Perform a dry run without making changes"`
	SkipThumbnails     bool   `arg:"--skip-thumbnails" help:"Skip thumbnail generation"`
	DeleteOriginals    bool   `arg:"--delete-originals" help:"Delete original files after successful import"`
	AutoEjectMacOS     bool   `arg:"--auto-eject-macos" help:"Automatically eject media after import on macOS (e.g., source drive)"`
	SidecarDefault     string `arg:"--sidecar-default" help:"Default action for unknown sidecar types (ignore/copy/delete)" default:"delete"`
}

// config holds the application configuration
type config struct {
	SourceDir          string                   `yaml:"source_directory"`
	DestDir            string                   `yaml:"destination_directory"`
	ConfigFile         string                   `yaml:"-"`
	OrganizeByDate     bool                     `yaml:"organize_by_date"`
	RenameByDateTime   bool                     `yaml:"rename_by_date_time"`
	ChecksumDuplicates bool                     `yaml:"checksum_duplicates"`
	Verbose            bool                     `yaml:"verbose"`
	DryRun             bool                     `yaml:"dry_run"`
	SkipThumbnails     bool                     `yaml:"skip_thumbnails"`
	DeleteOriginals    bool                     `yaml:"delete_originals"`
	AutoEjectMacOS     bool                     `yaml:"auto_eject_macos"`
	SidecarDefault     SidecarAction            `yaml:"sidecar_default"`
	Sidecars           map[string]SidecarAction `yaml:"sidecars"`
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
	cfg.ChecksumDuplicates = false
	cfg.Verbose = false
	cfg.DryRun = false
	cfg.SkipThumbnails = false
	cfg.DeleteOriginals = false
	cfg.AutoEjectMacOS = false
	cfg.SidecarDefault = SidecarDelete
	cfg.Sidecars = make(map[string]SidecarAction)
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

	// Check if destination directory's parent exists
	destParent := filepath.Dir(cfg.DestDir)
	if _, err := os.Stat(destParent); os.IsNotExist(err) {
		return fmt.Errorf("destination parent directory does not exist: %s", destParent)
	}

	// Validate sidecar default action
	if !isValidSidecarAction(cfg.SidecarDefault) {
		return fmt.Errorf("invalid sidecar default action: %q (must be ignore, copy, or delete)", cfg.SidecarDefault)
	}

	// Validate per-extension sidecar overrides
	for ext, action := range cfg.Sidecars {
		if !isValidSidecarAction(action) {
			return fmt.Errorf("invalid sidecar action for extension %q: %q (must be ignore, copy, or delete)", ext, action)
		}
	}

	return nil
}

// wasFlagProvided checks if a CLI flag was explicitly provided
func wasFlagProvided(flagName string) bool {
	for _, a := range os.Args[1:] {
		if a == flagName || strings.HasPrefix(a, flagName+"=") {
			return true
		}
	}
	return false
}

func run() error {
	// Create an instance of the config struct
	cfg := config{}

	// Set default values first
	if err := setDefaults(&cfg); err != nil {
		return fmt.Errorf("setting defaults: %w", err)
	}

	// Parse command-line arguments
	arg.MustParse(&args)

	// Apply config file path from command-line argument if provided
	if args.ConfigFile != "" {
		cfg.ConfigFile = args.ConfigFile
	}

	// Parse configuration file
	if err := parseConfigFile(&cfg); err != nil {
		return fmt.Errorf("parsing config file: %w", err)
	}

	// Override with command-line arguments
	if args.SourceDir != "" {
		cfg.SourceDir = args.SourceDir
	}
	if args.DestDir != "" {
		cfg.DestDir = args.DestDir
	}
	if wasFlagProvided("--organize-by-date") {
		cfg.OrganizeByDate = args.OrganizeByDate
	}
	if wasFlagProvided("--rename-by-date-time") {
		cfg.RenameByDateTime = args.RenameByDateTime
	}
	if wasFlagProvided("--checksum-duplicates") {
		cfg.ChecksumDuplicates = args.ChecksumDuplicates
	}
	if wasFlagProvided("-v") || wasFlagProvided("--verbose") {
		cfg.Verbose = args.Verbose
	}
	if wasFlagProvided("--dry-run") {
		cfg.DryRun = args.DryRun
	}
	if wasFlagProvided("--skip-thumbnails") {
		cfg.SkipThumbnails = args.SkipThumbnails
	}
	if wasFlagProvided("--delete-originals") {
		cfg.DeleteOriginals = args.DeleteOriginals
	}
	if wasFlagProvided("--auto-eject-macos") {
		cfg.AutoEjectMacOS = args.AutoEjectMacOS
	}
	if wasFlagProvided("--sidecar-default") {
		cfg.SidecarDefault = SidecarAction(args.SidecarDefault)
	}

	// Validate the configuration
	if err := validateConfig(&cfg); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Call the importMedia function
	if err := importMedia(cfg); err != nil {
		return fmt.Errorf("importing media: %w", err)
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
