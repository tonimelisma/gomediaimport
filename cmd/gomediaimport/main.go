package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexflint/go-arg"
	"gopkg.in/yaml.v3"
)

// version is set at build time via -ldflags or defaults to "dev"
var version = "dev"

// cliArgs holds the command-line arguments
type cliArgs struct {
	SourceDir            string      `arg:"--source" help:"Source directory for media files"`
	DestDir              string      `arg:"--dest" help:"Destination directory for imported media"`
	ConfigFile           string      `arg:"--config" help:"Path to config file"`
	OrganizeByDate       bool        `arg:"--organize-by-date" help:"Organize files by date"`
	RenameByDateTime     bool        `arg:"--rename-by-date-time" help:"Rename files by date and time"`
	ChecksumDuplicates   bool        `arg:"--checksum-duplicates" help:"Use checksums to identify duplicates (default)"`
	NoChecksumDuplicates bool        `arg:"--no-checksum-duplicates" help:"Disable checksum duplicate verification"`
	Verbose              bool        `arg:"-v,--verbose" help:"Enable verbose output"`
	Quiet                bool        `arg:"-q,--quiet" help:"Suppress all non-error output"`
	DryRun               bool        `arg:"--dry-run" help:"Perform a dry run without making changes"`
	DeleteOriginals      bool        `arg:"--delete-originals" help:"Delete imported originals and excluded source artifacts after successful import"`
	AutoEject            bool        `arg:"--auto-eject" help:"Automatically eject source media after successful import"`
	CheckDiskSpace       bool        `arg:"--check-disk-space" help:"Check for free disk space before importing" default:"true"`
	SidecarDefault       string      `arg:"--sidecar-default" help:"Default action for unknown sidecar types (ignore/copy/delete)" default:"delete"`
	Workers              int         `arg:"--workers" help:"Number of concurrent copy workers (0 = default of 4)"`
	Volumes              *volumesCmd `arg:"subcommand:volumes" help:"Manage remembered removable volume labels"`
}

// Version returns the version string for --version flag
func (cliArgs) Version() string {
	return "gomediaimport " + version
}

// config holds the application configuration
type config struct {
	SourceDir          string                           `yaml:"source_directory"`
	DestDir            string                           `yaml:"destination_directory"`
	ConfigFile         string                           `yaml:"-"`
	OrganizeByDate     bool                             `yaml:"organize_by_date"`
	RenameByDateTime   bool                             `yaml:"rename_by_date_time"`
	ChecksumDuplicates bool                             `yaml:"checksum_duplicates"`
	Verbose            bool                             `yaml:"verbose"`
	Quiet              bool                             `yaml:"quiet"`
	DryRun             bool                             `yaml:"dry_run"`
	DeleteOriginals    bool                             `yaml:"delete_originals"`
	AutoEject          bool                             `yaml:"auto_eject"`
	CheckDiskSpace     bool                             `yaml:"check_disk_space"`
	SidecarDefault     SidecarAction                    `yaml:"sidecar_default"`
	Sidecars           map[string]SidecarAction         `yaml:"sidecars"`
	Workers            int                              `yaml:"workers"`
	RemovableVolumes   map[string]removableVolumeConfig `yaml:"removable_volumes,omitempty"`
}

// setDefaults initializes the config with default values
func setDefaults(cfg *config) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %v", err)
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get user config directory: %v", err)
	}

	cfg.DestDir = filepath.Join(homeDir, "Pictures")
	cfg.ConfigFile = filepath.Join(configDir, "gomediaimport", "config.yaml")
	cfg.OrganizeByDate = false
	cfg.RenameByDateTime = false
	cfg.ChecksumDuplicates = true
	cfg.Verbose = false
	cfg.DryRun = false
	cfg.DeleteOriginals = false
	cfg.AutoEject = false
	cfg.CheckDiskSpace = true
	cfg.SidecarDefault = SidecarDelete
	cfg.Sidecars = make(map[string]SidecarAction)
	cfg.Workers = 0
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

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(cfg); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("failed to parse config file: %v", err)
	}

	return nil
}

// validateConfig checks if the configuration is valid
func validateCommonConfig(cfg *config) error {
	if cfg.DestDir == "" {
		return fmt.Errorf("destination directory is not specified")
	}

	// Check if destination directory's parent exists
	destParent := filepath.Dir(cfg.DestDir)
	if _, err := os.Stat(destParent); os.IsNotExist(err) {
		return fmt.Errorf("destination parent directory does not exist: %s", destParent)
	}

	// Validate workers count
	if cfg.Workers < 0 {
		return fmt.Errorf("workers must be non-negative, got %d", cfg.Workers)
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

	for label := range cfg.RemovableVolumes {
		if label == "" {
			return fmt.Errorf("removable volume label cannot be empty")
		}
	}

	return nil
}

// validateConfig checks if the configuration is valid for a single-source import.
func validateConfig(cfg *config) error {
	if err := validateCommonConfig(cfg); err != nil {
		return err
	}

	if cfg.SourceDir == "" {
		return fmt.Errorf("source directory is not specified")
	}

	// Check if source directory exists
	if _, err := os.Stat(cfg.SourceDir); os.IsNotExist(err) {
		return fmt.Errorf("source directory does not exist: %s", cfg.SourceDir)
	}

	return nil
}

// wasFlagProvided checks if a CLI flag was explicitly provided
func wasFlagProvided(osArgs []string, flagName string) bool {
	for _, a := range osArgs[1:] {
		if a == flagName || strings.HasPrefix(a, flagName+"=") {
			return true
		}
	}
	return false
}

// errExitClean is a sentinel error for clean exit (help/version)
var errExitClean = errors.New("clean exit")

func run(osArgs []string) error {
	// Create an instance of the config struct
	cfg := config{}

	// Set default values first
	if err := setDefaults(&cfg); err != nil {
		return fmt.Errorf("setting defaults: %w", err)
	}

	// Parse command-line arguments
	var parsedArgs cliArgs
	parser, err := arg.NewParser(arg.Config{}, &parsedArgs)
	if err != nil {
		return fmt.Errorf("creating argument parser: %w", err)
	}

	err = parser.Parse(osArgs[1:])
	switch {
	case errors.Is(err, arg.ErrHelp):
		writeHelp(parser, cfg)
		return errExitClean
	case errors.Is(err, arg.ErrVersion):
		fmt.Println(parsedArgs.Version())
		return errExitClean
	case err != nil:
		parser.WriteUsage(os.Stderr)
		return fmt.Errorf("parsing arguments: %w", err)
	}

	// Apply config file path from command-line argument if provided
	if parsedArgs.ConfigFile != "" {
		cfg.ConfigFile = parsedArgs.ConfigFile
	}

	if wasFlagProvided(osArgs, "--checksum-duplicates") && wasFlagProvided(osArgs, "--no-checksum-duplicates") {
		return fmt.Errorf("--checksum-duplicates and --no-checksum-duplicates cannot be used together")
	}

	// Parse configuration file
	if err := parseConfigFile(&cfg); err != nil {
		return fmt.Errorf("parsing config file: %w", err)
	}

	sourceProvided := parsedArgs.SourceDir != ""

	if parsedArgs.Volumes != nil {
		if parsedArgs.Volumes.Add != nil && parsedArgs.Volumes.Add.DestDir == "" && parsedArgs.DestDir != "" {
			parsedArgs.Volumes.Add.DestDir = parsedArgs.DestDir
		}
		if err := runVolumesCommand(parsedArgs.Volumes, cfg); err != nil {
			return err
		}
		return nil
	}

	// Override with command-line arguments
	if parsedArgs.SourceDir != "" {
		cfg.SourceDir = parsedArgs.SourceDir
	}
	if parsedArgs.DestDir != "" {
		cfg.DestDir = parsedArgs.DestDir
	}
	if wasFlagProvided(osArgs, "--organize-by-date") {
		cfg.OrganizeByDate = parsedArgs.OrganizeByDate
	}
	if wasFlagProvided(osArgs, "--rename-by-date-time") {
		cfg.RenameByDateTime = parsedArgs.RenameByDateTime
	}
	if wasFlagProvided(osArgs, "--checksum-duplicates") {
		cfg.ChecksumDuplicates = parsedArgs.ChecksumDuplicates
	}
	if wasFlagProvided(osArgs, "--no-checksum-duplicates") {
		cfg.ChecksumDuplicates = false
	}
	if wasFlagProvided(osArgs, "-v") || wasFlagProvided(osArgs, "--verbose") {
		cfg.Verbose = parsedArgs.Verbose
	}
	if wasFlagProvided(osArgs, "--dry-run") {
		cfg.DryRun = parsedArgs.DryRun
	}
	if wasFlagProvided(osArgs, "--delete-originals") {
		cfg.DeleteOriginals = parsedArgs.DeleteOriginals
	}
	if wasFlagProvided(osArgs, "--auto-eject") {
		cfg.AutoEject = parsedArgs.AutoEject
	}
	if wasFlagProvided(osArgs, "--check-disk-space") {
		cfg.CheckDiskSpace = parsedArgs.CheckDiskSpace
	}
	if wasFlagProvided(osArgs, "--sidecar-default") {
		cfg.SidecarDefault = SidecarAction(parsedArgs.SidecarDefault)
	}
	if wasFlagProvided(osArgs, "--workers") {
		cfg.Workers = parsedArgs.Workers
	}
	if wasFlagProvided(osArgs, "-q") || wasFlagProvided(osArgs, "--quiet") {
		cfg.Quiet = parsedArgs.Quiet
	}
	if cfg.Quiet {
		cfg.Verbose = false
	}

	if !sourceProvided && len(cfg.RemovableVolumes) > 0 {
		if err := validateCommonConfig(&cfg); err != nil {
			return fmt.Errorf("invalid configuration: %w", err)
		}
		if err := importConfiguredRemovableVolumes(cfg); err != nil {
			return fmt.Errorf("importing configured removable volumes: %w", err)
		}
		return nil
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

func writeHelp(parser *arg.Parser, cfg config) {
	parser.WriteHelp(os.Stdout)
	fmt.Fprintf(os.Stdout, "\nDefaults:\n  Config file: %s\n", cfg.ConfigFile)
}

func main() {
	if err := run(os.Args); err != nil {
		if errors.Is(err, errExitClean) {
			return
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
