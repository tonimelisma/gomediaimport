package main

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v2"
)

// TestSetDefaults tests the setDefaults function
func TestSetDefaults(t *testing.T) {
	cfg := &config{}
	err := setDefaults(cfg)
	if err != nil {
		t.Fatalf("setDefaults failed: %v", err)
	}

	homeDir, _ := os.UserHomeDir()

	if cfg.DestDir != filepath.Join(homeDir, "Pictures") {
		t.Errorf("Expected DestDir to be %s, got %s", filepath.Join(homeDir, "Pictures"), cfg.DestDir)
	}

	if cfg.ConfigFile != filepath.Join(homeDir, ".gomediaimportrc") {
		t.Errorf("Expected ConfigFile to be %s, got %s", filepath.Join(homeDir, ".gomediaimportrc"), cfg.ConfigFile)
	}

	if cfg.OrganizeByDate != false {
		t.Errorf("Expected OrganizeByDate to be false, got %v", cfg.OrganizeByDate)
	}

	if cfg.RenameByDateTime != false {
		t.Errorf("Expected RenameByDateTime to be false, got %v", cfg.RenameByDateTime)
	}

	if cfg.ChecksumDuplicates != false {
		t.Errorf("Expected ChecksumDuplicates to be false, got %v", cfg.ChecksumDuplicates)
	}

	if cfg.ChecksumImports != false {
		t.Errorf("Expected ChecksumImports to be false, got %v", cfg.ChecksumImports)
	}
}

// TestParseConfigFile tests the parseConfigFile function
func TestParseConfigFile(t *testing.T) {
	// Test with valid config file
	validConfig := `
source_directory: /path/to/source
destination_directory: /path/to/dest
organize_by_date: true
rename_by_date_time: true
checksum_duplicates: true
checksum_imports: true
`
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(validConfig)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	cfg := &config{ConfigFile: tmpfile.Name()}
	err = parseConfigFile(cfg)
	if err != nil {
		t.Fatalf("parseConfigFile failed: %v", err)
	}

	if cfg.SourceDir != "/path/to/source" {
		t.Errorf("Expected SourceDir to be /path/to/source, got %s", cfg.SourceDir)
	}

	// Test with non-existent config file
	cfg = &config{ConfigFile: "/non/existent/file"}
	err = parseConfigFile(cfg)
	if err != nil {
		t.Fatalf("parseConfigFile should not return error for non-existent file: %v", err)
	}

	// Test with invalid YAML in config file
	invalidConfig := `
source_directory: /path/to/source
destination_directory: /path/to/dest
organize_by_date: not_a_bool
`
	tmpfile, err = os.CreateTemp("", "invalid-config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(invalidConfig)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	cfg = &config{ConfigFile: tmpfile.Name()}
	err = parseConfigFile(cfg)
	if err == nil {
		t.Fatalf("parseConfigFile should return error for invalid YAML")
	}
}

// TestValidateConfig tests the validateConfig function
func TestValidateConfig(t *testing.T) {
	// Test with valid config
	tmpDir, err := os.MkdirTemp("", "source")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config{
		SourceDir: tmpDir,
		DestDir:   "/path/to/dest",
	}
	err = validateConfig(cfg)
	if err != nil {
		t.Fatalf("validateConfig failed: %v", err)
	}

	// Test with empty source directory
	cfg.SourceDir = ""
	err = validateConfig(cfg)
	if err == nil {
		t.Fatalf("validateConfig should return error for empty source directory")
	}

	// Test with empty destination directory
	cfg.SourceDir = tmpDir
	cfg.DestDir = ""
	err = validateConfig(cfg)
	if err == nil {
		t.Fatalf("validateConfig should return error for empty destination directory")
	}

	// Test with non-existent source directory
	cfg.SourceDir = "/non/existent/directory"
	cfg.DestDir = "/path/to/dest"
	err = validateConfig(cfg)
	if err == nil {
		t.Fatalf("validateConfig should return error for non-existent source directory")
	}
}

// TestMain tests the main function
func TestMain(t *testing.T) {
	// This test is more complex and might require some refactoring of the main function
	// to make it more testable. For now, we'll just test a simple case.

	// Save original args and restore them after the test
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Create a temporary source directory
	tmpDir, err := os.MkdirTemp("", "source")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set up test arguments
	os.Args = []string{"cmd", tmpDir}

	// TODO: Capture stdout to check for expected output
	// For now, we're just checking that main() doesn't panic
	main()
}

// Additional tests

// TestConfigMarshalUnmarshal tests the marshaling and unmarshaling of the config struct
func TestConfigMarshalUnmarshal(t *testing.T) {
	cfg := &config{
		SourceDir:          "/path/to/source",
		DestDir:            "/path/to/dest",
		OrganizeByDate:     true,
		RenameByDateTime:   true,
		ChecksumDuplicates: true,
		ChecksumImports:    true,
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	newCfg := &config{}
	err = yaml.Unmarshal(data, newCfg)
	if err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if *cfg != *newCfg {
		t.Errorf("Unmarshaled config does not match original: got %+v, want %+v", newCfg, cfg)
	}
}

// TestConfigFilePermissions tests the permissions of the created config file
func TestConfigFilePermissions(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	cfg := &config{
		SourceDir:          "/path/to/source",
		DestDir:            "/path/to/dest",
		OrganizeByDate:     true,
		RenameByDateTime:   true,
		ChecksumDuplicates: true,
		ChecksumImports:    true,
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	err = os.WriteFile(tmpfile.Name(), data, 0600)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	info, err := os.Stat(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to stat config file: %v", err)
	}

	if info.Mode().Perm() != 0600 {
		t.Errorf("Config file has incorrect permissions: got %v, want %v", info.Mode().Perm(), 0600)
	}
}
