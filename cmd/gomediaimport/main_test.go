package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
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

	if cfg.Workers != 0 {
		t.Errorf("Expected Workers to be 0, got %d", cfg.Workers)
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

	destDir := filepath.Join(tmpDir, "dest")
	cfg := &config{
		SourceDir:      tmpDir,
		DestDir:        destDir,
		SidecarDefault: SidecarDelete,
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
	cfg.DestDir = destDir
	err = validateConfig(cfg)
	if err == nil {
		t.Fatalf("validateConfig should return error for non-existent source directory")
	}
}

func TestValidateConfigNegativeWorkers(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "source")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config{
		SourceDir:      tmpDir,
		DestDir:        filepath.Join(tmpDir, "dest"),
		SidecarDefault: SidecarDelete,
		Workers:        -1,
	}
	err = validateConfig(cfg)
	if err == nil {
		t.Error("validateConfig should return error for negative workers")
	}
}

func TestValidateConfigDestination(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "source")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Dest parent doesn't exist
	cfg := &config{
		SourceDir:      tmpDir,
		DestDir:        "/non/existent/parent/dest",
		SidecarDefault: SidecarDelete,
	}
	err = validateConfig(cfg)
	if err == nil {
		t.Fatalf("validateConfig should return error for non-existent destination parent")
	}
}

// TestRun tests the run function
func TestRun(t *testing.T) {
	// Save original args and restore them after the test
	savedArgs := args
	defer func() { args = savedArgs }()
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

	// Test that run() completes without error
	if err := run(); err != nil {
		t.Errorf("run() returned error: %v", err)
	}
}

func TestRunSourceFromConfig(t *testing.T) {
	savedArgs := args
	defer func() { args = savedArgs }()
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Create a temporary source directory
	tmpDir, err := os.MkdirTemp("", "source")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a config file with source_directory set
	configContent := "source_directory: " + tmpDir + "\n"
	configFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp config file: %v", err)
	}
	defer os.Remove(configFile.Name())
	if _, err := configFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	if err := configFile.Close(); err != nil {
		t.Fatalf("Failed to close config file: %v", err)
	}

	// No positional arg — source comes from config file only
	os.Args = []string{"cmd", "--config", configFile.Name()}

	if err := run(); err != nil {
		t.Errorf("run() returned error when source dir is in config file: %v", err)
	}
}

func TestRunNoSourceAnywhere(t *testing.T) {
	savedArgs := args
	defer func() { args = savedArgs }()
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Create an empty config file (no source_directory)
	configFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp config file: %v", err)
	}
	defer os.Remove(configFile.Name())
	if _, err := configFile.Write([]byte("verbose: false\n")); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	if err := configFile.Close(); err != nil {
		t.Fatalf("Failed to close config file: %v", err)
	}

	// No positional arg, no source in config
	os.Args = []string{"cmd", "--config", configFile.Name()}

	err = run()
	if err == nil {
		t.Error("run() should return error when source dir is not provided anywhere")
	}
}

func TestRunInvalidSource(t *testing.T) {
	savedArgs := args
	defer func() { args = savedArgs }()
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"cmd", "/non/existent/source/dir"}

	err := run()
	if err == nil {
		t.Error("run() should return error for non-existent source directory")
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "copyfile-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "source.txt")
	dstPath := filepath.Join(tmpDir, "dest.txt")
	content := []byte("hello world test content")

	if err := os.WriteFile(srcPath, content, 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	if err := copyFile(srcPath, dstPath); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	got, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read dest file: %v", err)
	}

	if string(got) != string(content) {
		t.Errorf("copied content mismatch: got %q, want %q", got, content)
	}
}

func TestExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "exists-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	existingFile := filepath.Join(tmpDir, "exists.txt")
	if err := os.WriteFile(existingFile, []byte("data"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	found, err := exists(existingFile)
	if err != nil {
		t.Fatalf("exists() returned error for existing file: %v", err)
	}
	if !found {
		t.Error("exists() returned false for existing file")
	}

	missingFile := filepath.Join(tmpDir, "missing.txt")
	found, err = exists(missingFile)
	if err != nil {
		t.Fatalf("exists() returned error for missing file: %v", err)
	}
	if found {
		t.Error("exists() returned true for missing file")
	}
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
		SidecarDefault:     SidecarDelete,
		Sidecars:           map[string]SidecarAction{"xmp": SidecarCopy},
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

	if !reflect.DeepEqual(cfg, newCfg) {
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

func TestWasFlagProvided(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"cmd", "/some/dir", "--verbose", "--auto-eject-macos=false"}

	if !wasFlagProvided("--verbose") {
		t.Error("expected --verbose to be detected")
	}
	if !wasFlagProvided("--auto-eject-macos") {
		t.Error("expected --auto-eject-macos to be detected (=false form)")
	}
	if wasFlagProvided("--dry-run") {
		t.Error("expected --dry-run to NOT be detected")
	}
}

func TestAutoEjectMacOSConfiguration(t *testing.T) {
	createTempConfig := func(t *testing.T, content string) string {
		t.Helper()
		tmpfile, err := os.CreateTemp("", "config-autoeject-*.yaml")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		if _, err := tmpfile.Write([]byte(content)); err != nil {
			tmpfile.Close()
			os.Remove(tmpfile.Name())
			t.Fatalf("Failed to write to temp file: %v", err)
		}
		if err := tmpfile.Close(); err != nil {
			os.Remove(tmpfile.Name())
			t.Fatalf("Failed to close temp file: %v", err)
		}
		return tmpfile.Name()
	}

	t.Run("ParseFromConfigFile_True", func(t *testing.T) {
		tmpFileName := createTempConfig(t, "auto_eject_macos: true")
		defer os.Remove(tmpFileName)

		cfg := &config{}
		if err := setDefaults(cfg); err != nil {
			t.Fatalf("setDefaults failed: %v", err)
		}
		cfg.ConfigFile = tmpFileName
		if err := parseConfigFile(cfg); err != nil {
			t.Fatalf("parseConfigFile failed: %v", err)
		}
		if !cfg.AutoEjectMacOS {
			t.Errorf("Expected AutoEjectMacOS to be true from config file, got false")
		}
	})

	t.Run("ParseFromConfigFile_False", func(t *testing.T) {
		tmpFileName := createTempConfig(t, "auto_eject_macos: false")
		defer os.Remove(tmpFileName)

		cfg := &config{}
		if err := setDefaults(cfg); err != nil {
			t.Fatalf("setDefaults failed: %v", err)
		}
		cfg.ConfigFile = tmpFileName
		if err := parseConfigFile(cfg); err != nil {
			t.Fatalf("parseConfigFile failed: %v", err)
		}
		if cfg.AutoEjectMacOS {
			t.Errorf("Expected AutoEjectMacOS to be false from config file, got true")
		}
	})

	t.Run("CLITrueOverConfigFalse", func(t *testing.T) {
		tmpFileName := createTempConfig(t, "auto_eject_macos: false")
		defer os.Remove(tmpFileName)

		savedArgs := args
		defer func() { args = savedArgs }()

		oldOsArgs := os.Args
		defer func() { os.Args = oldOsArgs }()

		os.Args = []string{"cmd", "/tmp", "--auto-eject-macos"}
		args.AutoEjectMacOS = true
		args.ConfigFile = tmpFileName

		cfg := &config{}
		if err := setDefaults(cfg); err != nil {
			t.Fatalf("setDefaults failed: %v", err)
		}
		cfg.ConfigFile = args.ConfigFile
		if err := parseConfigFile(cfg); err != nil {
			t.Fatalf("parseConfigFile failed: %v", err)
		}
		if wasFlagProvided("--auto-eject-macos") {
			cfg.AutoEjectMacOS = args.AutoEjectMacOS
		}

		if !cfg.AutoEjectMacOS {
			t.Errorf("Expected AutoEjectMacOS to be true (CLI override), got false")
		}
	})

	t.Run("CLIFalseOverConfigTrue", func(t *testing.T) {
		tmpFileName := createTempConfig(t, "auto_eject_macos: true")
		defer os.Remove(tmpFileName)

		savedArgs := args
		defer func() { args = savedArgs }()

		oldOsArgs := os.Args
		defer func() { os.Args = oldOsArgs }()

		os.Args = []string{"cmd", "/tmp", "--auto-eject-macos=false"}
		args.AutoEjectMacOS = false
		args.ConfigFile = tmpFileName

		cfg := &config{}
		if err := setDefaults(cfg); err != nil {
			t.Fatalf("setDefaults failed: %v", err)
		}
		cfg.ConfigFile = args.ConfigFile
		if err := parseConfigFile(cfg); err != nil {
			t.Fatalf("parseConfigFile failed: %v", err)
		}
		if wasFlagProvided("--auto-eject-macos") {
			cfg.AutoEjectMacOS = args.AutoEjectMacOS
		}

		if cfg.AutoEjectMacOS {
			t.Errorf("Expected AutoEjectMacOS to be false (CLI override), got true")
		}
	})

	t.Run("DefaultValue", func(t *testing.T) {
		cfg := &config{}
		if err := setDefaults(cfg); err != nil {
			t.Fatalf("setDefaults failed: %v", err)
		}
		if cfg.AutoEjectMacOS {
			t.Errorf("Expected AutoEjectMacOS to be false (default), got true")
		}
	})
}

func TestSidecarConfigParsing(t *testing.T) {
	createTempConfig := func(t *testing.T, content string) string {
		t.Helper()
		tmpfile, err := os.CreateTemp("", "config-sidecar-*.yaml")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		if _, err := tmpfile.Write([]byte(content)); err != nil {
			tmpfile.Close()
			os.Remove(tmpfile.Name())
			t.Fatalf("Failed to write to temp file: %v", err)
		}
		if err := tmpfile.Close(); err != nil {
			os.Remove(tmpfile.Name())
			t.Fatalf("Failed to close temp file: %v", err)
		}
		return tmpfile.Name()
	}

	t.Run("ParseSidecarDefault", func(t *testing.T) {
		tmpFileName := createTempConfig(t, "sidecar_default: ignore")
		defer os.Remove(tmpFileName)

		cfg := &config{}
		if err := setDefaults(cfg); err != nil {
			t.Fatalf("setDefaults failed: %v", err)
		}
		cfg.ConfigFile = tmpFileName
		if err := parseConfigFile(cfg); err != nil {
			t.Fatalf("parseConfigFile failed: %v", err)
		}
		if cfg.SidecarDefault != SidecarIgnore {
			t.Errorf("Expected SidecarDefault to be %q, got %q", SidecarIgnore, cfg.SidecarDefault)
		}
	})

	t.Run("ParseSidecarOverrides", func(t *testing.T) {
		content := `
sidecar_default: delete
sidecars:
  xmp: copy
  srt: copy
  thm: ignore
`
		tmpFileName := createTempConfig(t, content)
		defer os.Remove(tmpFileName)

		cfg := &config{}
		if err := setDefaults(cfg); err != nil {
			t.Fatalf("setDefaults failed: %v", err)
		}
		cfg.ConfigFile = tmpFileName
		if err := parseConfigFile(cfg); err != nil {
			t.Fatalf("parseConfigFile failed: %v", err)
		}
		if cfg.SidecarDefault != SidecarDelete {
			t.Errorf("Expected SidecarDefault %q, got %q", SidecarDelete, cfg.SidecarDefault)
		}
		if cfg.Sidecars["xmp"] != SidecarCopy {
			t.Errorf("Expected xmp=copy, got %q", cfg.Sidecars["xmp"])
		}
		if cfg.Sidecars["thm"] != SidecarIgnore {
			t.Errorf("Expected thm=ignore, got %q", cfg.Sidecars["thm"])
		}
	})

	t.Run("DefaultSidecarValues", func(t *testing.T) {
		cfg := &config{}
		if err := setDefaults(cfg); err != nil {
			t.Fatalf("setDefaults failed: %v", err)
		}
		if cfg.SidecarDefault != SidecarDelete {
			t.Errorf("Expected default SidecarDefault %q, got %q", SidecarDelete, cfg.SidecarDefault)
		}
		if cfg.Sidecars == nil {
			t.Error("Expected Sidecars map to be initialized, got nil")
		}
	})
}

func TestValidateConfigInvalidSidecarAction(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "source")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	destDir := filepath.Join(tmpDir, "dest")

	t.Run("InvalidSidecarDefault", func(t *testing.T) {
		cfg := &config{
			SourceDir:      tmpDir,
			DestDir:        destDir,
			SidecarDefault: SidecarAction("bogus"),
		}
		err := validateConfig(cfg)
		if err == nil {
			t.Error("Expected error for invalid sidecar default, got nil")
		}
	})

	t.Run("InvalidSidecarOverride", func(t *testing.T) {
		cfg := &config{
			SourceDir:      tmpDir,
			DestDir:        destDir,
			SidecarDefault: SidecarDelete,
			Sidecars:       map[string]SidecarAction{"xmp": SidecarAction("bogus")},
		}
		err := validateConfig(cfg)
		if err == nil {
			t.Error("Expected error for invalid sidecar override action, got nil")
		}
	})
}
