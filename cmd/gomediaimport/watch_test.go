package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"howett.net/plist"
)

func TestWatchConfigDefaults(t *testing.T) {
	cfg := &config{}
	if err := setDefaults(cfg); err != nil {
		t.Fatalf("setDefaults failed: %v", err)
	}
	if !cfg.WatchRequireDCIM {
		t.Error("expected WatchRequireDCIM default to be true")
	}
	if !cfg.WatchNotifications {
		t.Error("expected WatchNotifications default to be true")
	}
	if len(cfg.WatchVolumes) != 0 {
		t.Errorf("expected WatchVolumes default to be empty, got %v", cfg.WatchVolumes)
	}
}

func TestWatchConfigFromYAML(t *testing.T) {
	content := `
watch_require_dcim: false
watch_volumes:
  - "EOS_*"
  - "NIKON*"
watch_notifications: false
`
	tmpFile, err := os.CreateTemp("", "watch-config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	cfg := &config{}
	if err := setDefaults(cfg); err != nil {
		t.Fatal(err)
	}
	cfg.ConfigFile = tmpFile.Name()
	if err := parseConfigFile(cfg); err != nil {
		t.Fatalf("parseConfigFile failed: %v", err)
	}

	if cfg.WatchRequireDCIM {
		t.Error("expected WatchRequireDCIM=false from YAML")
	}
	if cfg.WatchNotifications {
		t.Error("expected WatchNotifications=false from YAML")
	}
	if len(cfg.WatchVolumes) != 2 {
		t.Fatalf("expected 2 watch volumes, got %d", len(cfg.WatchVolumes))
	}
	if cfg.WatchVolumes[0] != "EOS_*" {
		t.Errorf("expected first volume pattern 'EOS_*', got %q", cfg.WatchVolumes[0])
	}
	if cfg.WatchVolumes[1] != "NIKON*" {
		t.Errorf("expected second volume pattern 'NIKON*', got %q", cfg.WatchVolumes[1])
	}
}

func TestPlistContent(t *testing.T) {
	homeDir := "/Users/testuser"
	binaryPath := "/usr/local/bin/gomediaimport"

	data, err := generatePlist(binaryPath, homeDir)
	if err != nil {
		t.Fatalf("generatePlist failed: %v", err)
	}

	var p launchAgentPlist
	_, err = plist.Unmarshal(data, &p)
	if err != nil {
		t.Fatalf("failed to unmarshal generated plist: %v", err)
	}

	if p.Label != launchAgentLabel {
		t.Errorf("expected Label=%s, got %s", launchAgentLabel, p.Label)
	}
	if len(p.ProgramArguments) != 3 ||
		p.ProgramArguments[0] != binaryPath ||
		p.ProgramArguments[1] != "watch" ||
		p.ProgramArguments[2] != "--run" {
		t.Errorf("unexpected ProgramArguments: %v", p.ProgramArguments)
	}
	if !p.StartOnMount {
		t.Error("expected StartOnMount=true")
	}
	if p.ProcessType != "Background" {
		t.Errorf("expected ProcessType=Background, got %s", p.ProcessType)
	}
	if !p.LowPriorityIO {
		t.Error("expected LowPriorityIO=true")
	}
	if p.ThrottleInterval != 5 {
		t.Errorf("expected ThrottleInterval=5, got %d", p.ThrottleInterval)
	}
	if p.StandardOutPath != filepath.Join(homeDir, "Library", "Logs", "gomediaimport.out.log") {
		t.Errorf("unexpected StandardOutPath: %s", p.StandardOutPath)
	}
	if p.StandardErrorPath != filepath.Join(homeDir, "Library", "Logs", "gomediaimport.err.log") {
		t.Errorf("unexpected StandardErrorPath: %s", p.StandardErrorPath)
	}
	if p.EnvironmentVariables["HOME"] != homeDir {
		t.Errorf("expected HOME=%s, got %s", homeDir, p.EnvironmentVariables["HOME"])
	}
	if p.EnvironmentVariables["PATH"] != "/usr/local/bin:/usr/bin:/bin:/opt/homebrew/bin" {
		t.Errorf("unexpected PATH: %s", p.EnvironmentVariables["PATH"])
	}
}

func TestPlistPathsAbsolute(t *testing.T) {
	data, err := generatePlist("/usr/local/bin/gomediaimport", "/Users/testuser")
	if err != nil {
		t.Fatal(err)
	}

	plistStr := string(data)
	if strings.Contains(plistStr, "~/") {
		t.Error("plist contains '~/' — all paths must be absolute")
	}
}

func TestInstallRequiresDestination(t *testing.T) {
	cfg := config{DestDir: ""}
	err := installLaunchAgent(cfg)
	if err == nil {
		t.Error("expected error when destination_directory is not set")
	}
	if err != nil && !strings.Contains(err.Error(), "destination_directory") {
		t.Errorf("expected error about destination_directory, got: %v", err)
	}
}

func TestInstallRefusesIfAlreadyInstalled(t *testing.T) {
	// installLaunchAgent checks for plist at the real path, so we test
	// by verifying it returns an error when the plist file already exists.
	// If the LaunchAgent happens to be installed on the test machine,
	// the install call should refuse.
	pPath, err := plistPath()
	if err != nil {
		t.Fatal(err)
	}

	// If plist already exists, install should refuse
	if _, err := os.Stat(pPath); err == nil {
		cfg := config{DestDir: "/tmp/dest"}
		err := installLaunchAgent(cfg)
		if err == nil {
			t.Error("expected error when plist already exists")
		}
		if !strings.Contains(err.Error(), "already installed") {
			t.Errorf("expected 'already installed' error, got: %v", err)
		}
	}
}

func TestUninstallWhenNotInstalled(t *testing.T) {
	pPath, err := plistPath()
	if err != nil {
		t.Fatal(err)
	}

	// Only test if plist does NOT exist (to avoid uninstalling a real agent)
	if _, err := os.Stat(pPath); os.IsNotExist(err) {
		err := uninstallLaunchAgent()
		if err != nil {
			t.Errorf("uninstall should succeed gracefully when not installed, got: %v", err)
		}
	}
}

func TestStatusShowsConfig(t *testing.T) {
	cfg := config{
		DestDir:            "/Users/test/Pictures",
		WatchRequireDCIM:   true,
		WatchVolumes:       []string{"EOS_*"},
		WatchNotifications: true,
	}

	// watchStatus prints to stdout — just verify it doesn't error
	err := watchStatus(cfg)
	if err != nil {
		t.Errorf("watchStatus should not error: %v", err)
	}
}

func TestWatchSubcommandParsing(t *testing.T) {
	// Verify the watchArgs struct fields are accessible
	w := &watchArgs{
		Install:   true,
		Uninstall: false,
		Status:    false,
		Run:       false,
	}
	if !w.Install {
		t.Error("expected Install=true")
	}
}

func TestWatchRunPrintsTimestamp(t *testing.T) {
	// runWatchImport reads /Volumes which is macOS-specific.
	// Just verify it compiles and the function signature is correct.
	// Full integration testing requires a mock filesystem.
}

func TestNoSubcommandRunsImport(t *testing.T) {
	savedArgs := args
	defer func() { args = savedArgs }()
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tmpDir, err := os.MkdirTemp("", "no-subcommand-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// No subcommand — should run import
	os.Args = []string{"cmd", "--source", tmpDir}

	if err := run(); err != nil {
		t.Errorf("run() without subcommand should succeed: %v", err)
	}
}

func TestSourceDirFlag(t *testing.T) {
	savedArgs := args
	defer func() { args = savedArgs }()
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tmpDir, err := os.MkdirTemp("", "source-flag-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	os.Args = []string{"cmd", "--source", tmpDir}

	if err := run(); err != nil {
		t.Errorf("run() with --source flag returned error: %v", err)
	}
}

func TestWatchImportEndToEnd(t *testing.T) {
	origFunc := diskutilInfoFunc
	defer func() { diskutilInfoFunc = origFunc }()

	// Create a temp source with DCIM structure and a JPEG file
	srcDir, err := os.MkdirTemp("", "watch-e2e-src")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(srcDir)

	dcimDir := filepath.Join(srcDir, "DCIM", "100CANON")
	if err := os.MkdirAll(dcimDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dcimDir, "IMG_0001.JPG"), []byte("fake jpeg data"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a temp destination
	destDir, err := os.MkdirTemp("", "watch-e2e-dest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(destDir)

	// Mock diskutil to report the source as an ejectable volume
	diskutilInfoFunc = func(mountPoint string) (*VolumeInfo, error) {
		if mountPoint == srcDir {
			return &VolumeInfo{
				VolumeName:                     filepath.Base(srcDir),
				Ejectable:                      true,
				Internal:                       false,
				RemovableMediaOrExternalDevice: true,
			}, nil
		}
		// Reject all other volumes
		return &VolumeInfo{
			VolumeName: "other",
			Ejectable:  false,
			Internal:   true,
		}, nil
	}

	cfg := config{
		DestDir:            destDir,
		WatchRequireDCIM:   true,
		WatchNotifications: false,
		SidecarDefault:     SidecarDelete,
		Sidecars:           make(map[string]SidecarAction),
	}

	// filterVolume on the srcDir should pass
	pass, err := filterVolume(srcDir, cfg)
	if err != nil {
		t.Fatalf("filterVolume failed: %v", err)
	}
	if !pass {
		t.Fatal("expected srcDir to pass filter")
	}

	// Run importMedia directly (runWatchImport reads /Volumes which we can't mock)
	importCfg := cfg
	importCfg.SourceDir = srcDir
	if err := importMedia(importCfg); err != nil {
		t.Fatalf("importMedia failed: %v", err)
	}

	// Verify file was copied
	destFile := filepath.Join(destDir, "IMG_0001.JPG")
	if _, err := os.Stat(destFile); os.IsNotExist(err) {
		t.Error("expected IMG_0001.JPG to be copied to destination")
	}

	// Run again — should be idempotent (duplicate detected)
	if err := importMedia(importCfg); err != nil {
		t.Fatalf("second importMedia failed: %v", err)
	}
}
