package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"howett.net/plist"
)

const launchAgentLabel = "com.github.tonimelisma.gomediaimport"

type launchAgentPlist struct {
	Label                string            `plist:"Label"`
	ProgramArguments     []string          `plist:"ProgramArguments"`
	StartOnMount         bool              `plist:"StartOnMount"`
	ProcessType          string            `plist:"ProcessType"`
	LowPriorityIO       bool              `plist:"LowPriorityIO"`
	ThrottleInterval     int               `plist:"ThrottleInterval"`
	StandardOutPath      string            `plist:"StandardOutPath"`
	StandardErrorPath    string            `plist:"StandardErrorPath"`
	EnvironmentVariables map[string]string `plist:"EnvironmentVariables"`
}

func runWatch(cfg config) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("watch mode is only supported on macOS")
	}
	switch {
	case args.Watch.Install:
		return installLaunchAgent(cfg)
	case args.Watch.Uninstall:
		return uninstallLaunchAgent()
	case args.Watch.Status:
		return watchStatus(cfg)
	case args.Watch.Run:
		return runWatchImport(cfg)
	default:
		return fmt.Errorf("specify --install, --uninstall, or --status")
	}
}

func plistPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, "Library", "LaunchAgents", launchAgentLabel+".plist"), nil
}

func generatePlist(binaryPath, homeDir string) ([]byte, error) {
	p := launchAgentPlist{
		Label:            launchAgentLabel,
		ProgramArguments: []string{binaryPath, "watch", "--run"},
		StartOnMount:     true,
		ProcessType:      "Background",
		LowPriorityIO:   true,
		ThrottleInterval: 5,
		StandardOutPath:  filepath.Join(homeDir, "Library", "Logs", "gomediaimport.out.log"),
		StandardErrorPath: filepath.Join(homeDir, "Library", "Logs", "gomediaimport.err.log"),
		EnvironmentVariables: map[string]string{
			"HOME": homeDir,
			"PATH": "/usr/local/bin:/usr/bin:/bin:/opt/homebrew/bin",
		},
	}
	return plist.MarshalIndent(p, plist.XMLFormat, "\t")
}

func installLaunchAgent(cfg config) error {
	pPath, err := plistPath()
	if err != nil {
		return err
	}

	// Refuse if already installed
	if _, err := os.Stat(pPath); err == nil {
		return fmt.Errorf("LaunchAgent already installed at %s\nRun 'gomediaimport watch --uninstall' first", pPath)
	}

	// Require destination directory
	if cfg.DestDir == "" {
		return fmt.Errorf("destination_directory must be set in ~/.gomediaimportrc before installing watch mode")
	}

	// Resolve binary path
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("failed to resolve symlinks for executable: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Generate plist
	data, err := generatePlist(exePath, homeDir)
	if err != nil {
		return fmt.Errorf("failed to generate plist: %w", err)
	}

	// Write plist file
	if err := os.MkdirAll(filepath.Dir(pPath), 0755); err != nil {
		return fmt.Errorf("failed to create LaunchAgents directory: %w", err)
	}
	if err := os.WriteFile(pPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write plist: %w", err)
	}

	// Validate with plutil
	cmd := exec.Command("plutil", "-lint", pPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		_ = os.Remove(pPath) // Clean up invalid plist
		return fmt.Errorf("plist validation failed: %s", output)
	}

	// Bootstrap the LaunchAgent
	uid := fmt.Sprintf("%d", os.Getuid())
	cmd = exec.Command("launchctl", "bootstrap", "gui/"+uid, pPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to bootstrap LaunchAgent: %s", output)
	}

	fmt.Printf("LaunchAgent installed successfully.\n")
	fmt.Printf("  Binary: %s\n", exePath)
	fmt.Printf("  Destination: %s\n", cfg.DestDir)
	fmt.Printf("  Plist: %s\n", pPath)
	return nil
}

func uninstallLaunchAgent() error {
	pPath, err := plistPath()
	if err != nil {
		return err
	}

	// Check if installed
	if _, err := os.Stat(pPath); os.IsNotExist(err) {
		fmt.Println("LaunchAgent is not installed.")
		return nil
	}

	// Bootout the LaunchAgent
	uid := fmt.Sprintf("%d", os.Getuid())
	cmd := exec.Command("launchctl", "bootout", "gui/"+uid+"/"+launchAgentLabel)
	if output, err := cmd.CombinedOutput(); err != nil {
		// Don't fail if bootout fails (agent might not be loaded)
		fmt.Fprintf(os.Stderr, "Warning: launchctl bootout: %s", output)
	}

	// Remove plist file
	if err := os.Remove(pPath); err != nil {
		return fmt.Errorf("failed to remove plist: %w", err)
	}

	fmt.Println("LaunchAgent uninstalled successfully.")
	return nil
}

func watchStatus(cfg config) error {
	pPath, err := plistPath()
	if err != nil {
		return err
	}

	_, statErr := os.Stat(pPath)
	installed := statErr == nil

	if installed {
		fmt.Println("Watch status: installed")
		fmt.Printf("  Plist: %s\n", pPath)
	} else {
		fmt.Println("Watch status: not installed")
	}

	fmt.Printf("\nWatch configuration:\n")
	fmt.Printf("  Destination directory: %s\n", cfg.DestDir)
	fmt.Printf("  Require DCIM folder: %v\n", cfg.WatchRequireDCIM)
	if len(cfg.WatchVolumes) > 0 {
		fmt.Printf("  Volume allowlist: %v\n", cfg.WatchVolumes)
	} else {
		fmt.Printf("  Volume allowlist: (all volumes)\n")
	}
	fmt.Printf("  Notifications: %v\n", cfg.WatchNotifications)

	return nil
}

func runWatchImport(cfg config) error {
	fmt.Printf("[%s] Watch import triggered\n", time.Now().Format("2006-01-02 15:04:05"))

	entries, err := os.ReadDir("/Volumes")
	if err != nil {
		return fmt.Errorf("failed to read /Volumes: %w", err)
	}

	var firstErr error
	importCount := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		mountPoint := filepath.Join("/Volumes", entry.Name())

		pass, err := filterVolume(mountPoint, cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: error filtering volume %s: %v\n", entry.Name(), err)
			continue
		}
		if !pass {
			continue
		}

		fmt.Printf("Importing from volume: %s\n", entry.Name())
		if cfg.WatchNotifications {
			sendNotification("Go Media Import", fmt.Sprintf("Camera card detected: %s — importing to %s...", entry.Name(), cfg.DestDir))
		}

		importCfg := cfg
		importCfg.SourceDir = mountPoint

		if err := validateConfig(&importCfg); err != nil {
			errMsg := fmt.Sprintf("invalid config for volume %s: %v", entry.Name(), err)
			fmt.Fprintf(os.Stderr, "Error: %s\n", errMsg)
			if cfg.WatchNotifications {
				sendNotification("Go Media Import", fmt.Sprintf("Import failed for %s: %s", entry.Name(), err))
			}
			if firstErr == nil {
				firstErr = fmt.Errorf("%s", errMsg)
			}
			continue
		}

		if err := importMedia(importCfg); err != nil {
			errMsg := fmt.Sprintf("import failed for %s: %v", entry.Name(), err)
			fmt.Fprintf(os.Stderr, "Error: %s\n", errMsg)
			if cfg.WatchNotifications {
				sendNotification("Go Media Import", fmt.Sprintf("Import failed for %s: %s", entry.Name(), err))
			}
			if firstErr == nil {
				firstErr = fmt.Errorf("%s", errMsg)
			}
			continue
		}

		importCount++
		if cfg.WatchNotifications {
			sendNotification("Go Media Import", fmt.Sprintf("Import complete from %s", entry.Name()))
		}
	}

	if importCount == 0 && firstErr == nil {
		fmt.Println("No matching volumes found.")
	}

	return firstErr
}
