package main

import (
	"fmt"
	"os"
	"os/exec"
)

// sendNotification sends a macOS notification using osascript.
// It is fire-and-forget — errors are logged to stderr but do not stop execution.
func sendNotification(title, message string) {
	script := fmt.Sprintf(`display notification %q with title %q`, message, title)
	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to start notification: %v\n", err)
		return
	}
	go func() {
		if err := cmd.Wait(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: notification failed: %v\n", err)
		}
	}()
}
