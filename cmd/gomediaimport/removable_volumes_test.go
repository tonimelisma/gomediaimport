package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func withMountedRemovableVolumes(t *testing.T, volumes []mountedRemovableVolume) {
	t.Helper()
	original := mountedRemovableVolumes
	mountedRemovableVolumes = func() ([]mountedRemovableVolume, error) {
		return append([]mountedRemovableVolume(nil), volumes...), nil
	}
	t.Cleanup(func() { mountedRemovableVolumes = original })
}

func withImportMediaRunner(t *testing.T, runner func(config) error) {
	t.Helper()
	original := importMediaRunner
	importMediaRunner = runner
	t.Cleanup(func() { importMediaRunner = original })
}

func TestRemovableVolumesConfigParsing(t *testing.T) {
	content := `
destination_directory: /default/dest
removable_volumes:
  SOFIA: {}
  "4152150790":
    destination_directory: /custom/dest
`
	var cfg config
	if err := yaml.Unmarshal([]byte(content), &cfg); err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}

	if _, ok := cfg.RemovableVolumes["SOFIA"]; !ok {
		t.Fatal("expected SOFIA removable volume entry")
	}
	if got := cfg.RemovableVolumes["4152150790"].DestDir; got != "/custom/dest" {
		t.Fatalf("got destination %q, want /custom/dest", got)
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}
	if !strings.Contains(string(data), "removable_volumes:") {
		t.Fatalf("expected marshaled config to include removable_volumes, got:\n%s", string(data))
	}
}

func TestListRemovableVolumes(t *testing.T) {
	withMountedRemovableVolumes(t, []mountedRemovableVolume{
		{Label: "UNSAVED", MountPath: "/Volumes/UNSAVED"},
		{Label: "SOFIA", MountPath: "/Volumes/SOFIA"},
		{Label: "CAM", MountPath: "/Volumes/CAM"},
	})

	cfg := config{
		DestDir: "/default/dest",
		RemovableVolumes: map[string]removableVolumeConfig{
			"SOFIA": {},
			"CAM":   {DestDir: "/custom/dest"},
		},
	}

	var out bytes.Buffer
	if err := listRemovableVolumes(cfg, &out); err != nil {
		t.Fatalf("listRemovableVolumes failed: %v", err)
	}

	output := out.String()
	for _, want := range []string{"ID", "LABEL", "SOURCE", "SAVED", "DESTINATION", "SOFIA", "/Volumes/SOFIA", "yes", "/default/dest", "CAM", "/custom/dest", "UNSAVED", "no"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, output)
		}
	}
}

func TestAddRemovableVolume(t *testing.T) {
	t.Run("AddsCurrentLabelWithDestinationAndPreservesUnknownKeys", func(t *testing.T) {
		destDir := t.TempDir()
		configPath := filepath.Join(t.TempDir(), "config.yaml")
		if err := os.WriteFile(configPath, []byte("destination_directory: "+destDir+"\nchecksum_imports: true\n"), 0600); err != nil {
			t.Fatal(err)
		}
		withMountedRemovableVolumes(t, []mountedRemovableVolume{{Label: "SOFIA", MountPath: "/Volumes/SOFIA"}})

		customDest := filepath.Join(destDir, "Sofia")
		if err := run([]string{"cmd", "--config", configPath, "volumes", "add", "SOFIA", "--dest", customDest}); err != nil {
			t.Fatalf("run volumes add failed: %v", err)
		}

		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		for _, want := range []string{"checksum_imports: true", "removable_volumes:", "SOFIA:", "destination_directory: " + customDest} {
			if !strings.Contains(text, want) {
				t.Fatalf("expected config to contain %q, got:\n%s", want, text)
			}
		}
	})

	t.Run("RejectsUnattachedLabel", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "config.yaml")
		if err := os.WriteFile(configPath, []byte("destination_directory: "+t.TempDir()+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		withMountedRemovableVolumes(t, []mountedRemovableVolume{{Label: "SOFIA", MountPath: "/Volumes/SOFIA"}})

		err := run([]string{"cmd", "--config", configPath, "volumes", "add", "MISSING"})
		if err == nil {
			t.Fatal("expected unattached label to fail")
		}
		if !strings.Contains(err.Error(), "not currently mounted") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("AcceptsNumericID", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "config.yaml")
		if err := os.WriteFile(configPath, []byte("destination_directory: "+t.TempDir()+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		withMountedRemovableVolumes(t, []mountedRemovableVolume{{Label: "SOFIA", MountPath: "/Volumes/SOFIA"}})

		if err := run([]string{"cmd", "--config", configPath, "volumes", "add", "1"}); err != nil {
			t.Fatalf("run volumes add by ID failed: %v", err)
		}

		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "SOFIA: {}") {
			t.Fatalf("expected config to contain SOFIA empty map, got:\n%s", string(data))
		}
	})

	t.Run("LeavesExistingDestinationWhenNoDestinationProvided", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "config.yaml")
		existingDest := filepath.Join(t.TempDir(), "existing")
		content := "destination_directory: " + t.TempDir() + "\nremovable_volumes:\n  SOFIA:\n    destination_directory: " + existingDest + "\n"
		if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		withMountedRemovableVolumes(t, []mountedRemovableVolume{{Label: "SOFIA", MountPath: "/Volumes/SOFIA"}})

		if err := run([]string{"cmd", "--config", configPath, "volumes", "add", "SOFIA"}); err != nil {
			t.Fatalf("run volumes add failed: %v", err)
		}

		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "destination_directory: "+existingDest) {
			t.Fatalf("expected existing destination to remain, got:\n%s", string(data))
		}
	})
}

func TestImportConfiguredRemovableVolumes(t *testing.T) {
	defaultDest := t.TempDir()
	customDest := t.TempDir()
	camMountA := t.TempDir()
	camMountB := t.TempDir()
	sofiaMount := t.TempDir()
	withMountedRemovableVolumes(t, []mountedRemovableVolume{
		{Label: "SOFIA", MountPath: sofiaMount},
		{Label: "CAM", MountPath: camMountB},
		{Label: "UNSAVED", MountPath: t.TempDir()},
		{Label: "CAM", MountPath: camMountA},
	})

	var calls []config
	withImportMediaRunner(t, func(cfg config) error {
		calls = append(calls, cfg)
		return nil
	})

	cfg := config{
		DestDir:        defaultDest,
		SidecarDefault: SidecarDelete,
		Sidecars:       map[string]SidecarAction{},
		RemovableVolumes: map[string]removableVolumeConfig{
			"CAM":   {},
			"SOFIA": {DestDir: customDest},
			"AWAY":  {},
		},
	}

	if err := importConfiguredRemovableVolumes(cfg); err != nil {
		t.Fatalf("importConfiguredRemovableVolumes failed: %v", err)
	}

	if len(calls) != 3 {
		t.Fatalf("got %d import calls, want 3", len(calls))
	}
	expected := []struct {
		source string
		dest   string
	}{
		{camMountA, defaultDest},
		{camMountB, defaultDest},
		{sofiaMount, customDest},
	}
	for i, want := range expected {
		if calls[i].SourceDir != want.source || calls[i].DestDir != want.dest {
			t.Fatalf("call %d got source=%q dest=%q, want source=%q dest=%q", i, calls[i].SourceDir, calls[i].DestDir, want.source, want.dest)
		}
	}
}

func TestImportConfiguredRemovableVolumesContinuesAfterFailure(t *testing.T) {
	defaultDest := t.TempDir()
	firstMount := t.TempDir()
	secondMount := t.TempDir()
	withMountedRemovableVolumes(t, []mountedRemovableVolume{
		{Label: "CAM", MountPath: firstMount},
		{Label: "SOFIA", MountPath: secondMount},
	})

	var calls []string
	withImportMediaRunner(t, func(cfg config) error {
		calls = append(calls, cfg.SourceDir)
		if cfg.SourceDir == firstMount {
			return errors.New("boom")
		}
		return nil
	})

	cfg := config{
		DestDir:        defaultDest,
		SidecarDefault: SidecarDelete,
		Sidecars:       map[string]SidecarAction{},
		RemovableVolumes: map[string]removableVolumeConfig{
			"CAM":   {},
			"SOFIA": {},
		},
	}

	err := importConfiguredRemovableVolumes(cfg)
	if err == nil {
		t.Fatal("expected joined import error")
	}
	if len(calls) != 2 {
		t.Fatalf("got %d calls, want 2", len(calls))
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected failure to include original error, got %v", err)
	}
}
