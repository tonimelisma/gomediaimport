package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexflint/go-arg"
	"howett.net/plist"
)

func TestWatchConfigDefaults(t *testing.T) {
	cfg := &config{}
	if err := setDefaults(cfg); err != nil {
		t.Fatalf("setDefaults failed: %v", err)
	}
	if !cfg.Watch.RequireDCIM {
		t.Error("expected Watch.RequireDCIM default to be true")
	}
	if !cfg.Watch.Notifications {
		t.Error("expected Watch.Notifications default to be true")
	}
	if len(cfg.Watch.Volumes) != 0 {
		t.Errorf("expected Watch.Volumes default to be empty, got %v", cfg.Watch.Volumes)
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

	if cfg.Watch.RequireDCIM {
		t.Error("expected Watch.RequireDCIM=false from YAML")
	}
	if cfg.Watch.Notifications {
		t.Error("expected Watch.Notifications=false from YAML")
	}
	if len(cfg.Watch.Volumes) != 2 {
		t.Fatalf("expected 2 watch volumes, got %d", len(cfg.Watch.Volumes))
	}
	if cfg.Watch.Volumes[0] != "EOS_*" {
		t.Errorf("expected first volume pattern 'EOS_*', got %q", cfg.Watch.Volumes[0])
	}
	if cfg.Watch.Volumes[1] != "NIKON*" {
		t.Errorf("expected second volume pattern 'NIKON*', got %q", cfg.Watch.Volumes[1])
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
	if p.EnvironmentVariables["PATH"] != "/usr/local/bin:/usr/bin:/usr/sbin:/bin:/opt/homebrew/bin" {
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
	tmpDir := t.TempDir()
	pPath := filepath.Join(tmpDir, "test.plist")
	cfg := config{DestDir: ""}
	err := installLaunchAgent(cfg, pPath)
	if err == nil {
		t.Error("expected error when destination_directory is not set")
	}
	if err != nil && !strings.Contains(err.Error(), "destination_directory") {
		t.Errorf("expected error about destination_directory, got: %v", err)
	}
}

func TestInstallRefusesIfAlreadyInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	pPath := filepath.Join(tmpDir, "test.plist")
	os.WriteFile(pPath, []byte("fake"), 0644)
	cfg := config{DestDir: "/tmp/dest"}
	err := installLaunchAgent(cfg, pPath)
	if err == nil {
		t.Error("expected error when plist already exists")
	}
	if !strings.Contains(err.Error(), "already installed") {
		t.Errorf("expected 'already installed' error, got: %v", err)
	}
}

func TestUninstallWhenNotInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	pPath := filepath.Join(tmpDir, "nonexistent.plist")

	err := uninstallLaunchAgent(pPath)
	if err != nil {
		t.Errorf("uninstall should succeed gracefully when not installed, got: %v", err)
	}
}

func TestStatusShowsConfig(t *testing.T) {
	tmpDir := t.TempDir()
	pPath := filepath.Join(tmpDir, "nonexistent.plist")
	cfg := config{
		DestDir: "/Users/test/Pictures",
		Watch: WatchConfig{
			RequireDCIM:   true,
			Volumes:       []string{"EOS_*"},
			Notifications: true,
		},
	}

	// watchStatus prints to stdout — just verify it doesn't error
	err := watchStatus(cfg, pPath)
	if err != nil {
		t.Errorf("watchStatus should not error: %v", err)
	}
}

func TestWatchStatusBinaryMissingWarning(t *testing.T) {
	tmpDir := t.TempDir()
	pPath := filepath.Join(tmpDir, "test.plist")

	// Create a plist with a non-existent binary
	data, err := generatePlist("/nonexistent/binary/gomediaimport", "/Users/testuser")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config{
		DestDir: "/tmp/dest",
		Watch:   WatchConfig{RequireDCIM: true, Notifications: true},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	watchErr := watchStatus(cfg, pPath)

	w.Close()
	os.Stdout = oldStdout

	if watchErr != nil {
		t.Fatalf("watchStatus should not error: %v", watchErr)
	}

	output := make([]byte, 4096)
	n, _ := r.Read(output)
	outputStr := string(output[:n])

	if !strings.Contains(outputStr, "WARNING: binary not found") {
		t.Errorf("expected output to contain 'WARNING: binary not found', got:\n%s", outputStr)
	}
}

func TestWatchSubcommandParsing(t *testing.T) {
	// Verify that the watch subcommand is parsed correctly via arg.NewParser
	var parsedArgs cliArgs
	p, err := arg.NewParser(arg.Config{}, &parsedArgs)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Parse([]string{"watch", "--install"}); err != nil {
		t.Fatalf("failed to parse watch --install: %v", err)
	}
	if parsedArgs.Watch == nil {
		t.Fatal("expected Watch subcommand to be set")
	}
	if !parsedArgs.Watch.Install {
		t.Error("expected Install=true")
	}
}

func TestNoSubcommandRunsImport(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "no-subcommand-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// No subcommand — should run import
	if err := run([]string{"cmd", "--source", tmpDir, "--config", emptyConfigFile(t)}); err != nil {
		t.Errorf("run() without subcommand should succeed: %v", err)
	}
}

func TestSourceDirFlag(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "source-flag-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	if err := run([]string{"cmd", "--source", tmpDir, "--config", emptyConfigFile(t)}); err != nil {
		t.Errorf("run() with --source flag returned error: %v", err)
	}
}

func TestWatchImportEndToEnd(t *testing.T) {
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
	mockFn := func(mountPoint string) (*VolumeInfo, error) {
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
		DestDir:        destDir,
		SidecarDefault: SidecarDelete,
		Sidecars:       make(map[string]SidecarAction),
		Watch: WatchConfig{
			RequireDCIM:   true,
			Notifications: false,
		},
	}

	// filterVolume on the srcDir should pass
	pass, err := filterVolume(srcDir, cfg, mockFn)
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

func TestRunWatchImportScansVolumes(t *testing.T) {
	// Create temp "volumes" dir with a subdirectory simulating a camera card
	volumesDir, err := os.MkdirTemp("", "volumes")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(volumesDir)

	cardDir := filepath.Join(volumesDir, "CARD")
	if err := os.MkdirAll(filepath.Join(cardDir, "DCIM"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cardDir, "DCIM", "IMG_0001.JPG"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	destDir, err := os.MkdirTemp("", "dest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(destDir)

	mockFn := func(mp string) (*VolumeInfo, error) {
		return &VolumeInfo{
			VolumeName:                     filepath.Base(mp),
			Ejectable:                      true,
			Internal:                       false,
			RemovableMediaOrExternalDevice: true,
		}, nil
	}

	cfg := config{
		DestDir:        destDir,
		SidecarDefault: SidecarDelete,
		Sidecars:       make(map[string]SidecarAction),
		Watch: WatchConfig{
			RequireDCIM:   true,
			Notifications: false,
		},
	}

	err = runWatchImport(cfg, volumesDir, mockFn)
	if err != nil {
		t.Fatalf("runWatchImport failed: %v", err)
	}

	// Verify file was copied
	if _, err := os.Stat(filepath.Join(destDir, "IMG_0001.JPG")); os.IsNotExist(err) {
		t.Error("expected file to be imported")
	}
}

func TestRunWatchImportCollectsAllErrors(t *testing.T) {
	// Create temp volumes dir with two subdirectories
	volumesDir, err := os.MkdirTemp("", "volumes-errs")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(volumesDir)

	// Create two fake volumes with DCIM
	for _, name := range []string{"CARD_A", "CARD_B"} {
		cardDir := filepath.Join(volumesDir, name)
		if err := os.MkdirAll(filepath.Join(cardDir, "DCIM"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(cardDir, "DCIM", "IMG.JPG"), []byte("data"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	mockFn := func(mp string) (*VolumeInfo, error) {
		return &VolumeInfo{
			VolumeName:                     filepath.Base(mp),
			Ejectable:                      true,
			Internal:                       false,
			RemovableMediaOrExternalDevice: true,
		}, nil
	}

	// Set DestDir to a path whose parent doesn't exist, so validateConfig fails for both
	cfg := config{
		DestDir:        "/nonexistent-parent/dest",
		SidecarDefault: SidecarDelete,
		Sidecars:       make(map[string]SidecarAction),
		Watch: WatchConfig{
			RequireDCIM:   true,
			Notifications: false,
		},
	}

	err = runWatchImport(cfg, volumesDir, mockFn)
	if err == nil {
		t.Fatal("expected error from runWatchImport, got nil")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "CARD_A") {
		t.Errorf("expected error to mention CARD_A, got: %s", errStr)
	}
	if !strings.Contains(errStr, "CARD_B") {
		t.Errorf("expected error to mention CARD_B, got: %s", errStr)
	}
}

func TestRunWatchImportVerboseLogging(t *testing.T) {
	volumesDir := t.TempDir()

	// Create a volume that will be filtered out (no DCIM)
	os.MkdirAll(filepath.Join(volumesDir, "USB_DRIVE"), 0755)

	mockFn := func(mp string) (*VolumeInfo, error) {
		return &VolumeInfo{
			VolumeName:                     filepath.Base(mp),
			Ejectable:                      true,
			Internal:                       false,
			RemovableMediaOrExternalDevice: true,
		}, nil
	}

	cfg := config{
		DestDir:        t.TempDir(),
		Verbose:        true,
		SidecarDefault: SidecarDelete,
		Sidecars:       make(map[string]SidecarAction),
		Watch: WatchConfig{
			RequireDCIM:   true,
			Notifications: false,
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	_ = runWatchImport(cfg, volumesDir, mockFn)

	w.Close()
	os.Stdout = oldStdout

	output := make([]byte, 8192)
	n, _ := r.Read(output)
	outputStr := string(output[:n])

	if !strings.Contains(outputStr, "Config:") {
		t.Errorf("expected verbose output to contain 'Config:', got:\n%s", outputStr)
	}
	if !strings.Contains(outputStr, "Scanning") {
		t.Errorf("expected verbose output to contain 'Scanning', got:\n%s", outputStr)
	}
	if !strings.Contains(outputStr, "Evaluating volume:") {
		t.Errorf("expected verbose output to contain 'Evaluating volume:', got:\n%s", outputStr)
	}
}
