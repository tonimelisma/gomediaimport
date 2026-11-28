package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Validates: GMI-009
func TestVersionFlagPrintsStampedVersion(t *testing.T) {
	binPath := filepath.Join(t.TempDir(), "gomediaimport")
	build := exec.Command("go", "build", "-trimpath", "-ldflags", "-X main.version=v9.8.7", "-o", binPath, ".")
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	cmd := exec.Command(binPath, "--version")
	cmd.Env = []string{"HOME=" + t.TempDir(), "XDG_CONFIG_HOME=" + t.TempDir()}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("--version: %v\n%s", err, out)
	}
	if got, want := string(out), "gomediaimport v9.8.7\n"; got != want {
		t.Errorf("--version printed %q, want %q", got, want)
	}
}

// Validates: GMI-009
func TestResolveVersion(t *testing.T) {
	if got := resolveVersion("v1.2.3"); got != "v1.2.3" {
		t.Errorf("resolveVersion(%q) = %q, want v1.2.3", "v1.2.3", got)
	}
	// Unstamped: the module version from the build info, or (devel); never empty.
	if got := resolveVersion(""); got == "" {
		t.Error("resolveVersion(\"\") returned an empty string")
	}
}

// Validates: GMI-082
func TestModulePathMatchesVersionSeries(t *testing.T) {
	series, err := os.ReadFile("../../VERSION")
	if err != nil {
		t.Fatalf("read VERSION: %v", err)
	}
	major, _, ok := strings.Cut(strings.TrimSpace(string(series)), ".")
	if !ok {
		t.Fatalf("VERSION %q is not MAJOR.MINOR", series)
	}
	gomod, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	first, _, _ := strings.Cut(string(gomod), "\n")
	want := "module github.com/tonimelisma/gomediaimport"
	if major != "0" && major != "1" {
		want += "/v" + major
	}
	// Go ignores vN tags (N >= 2) of a module whose path lacks the /vN suffix, so
	// `go install ...@latest` would install the last v1 release.
	if first != want {
		t.Errorf("go.mod starts with %q, want %q for series %s", first, want, strings.TrimSpace(string(series)))
	}
}
