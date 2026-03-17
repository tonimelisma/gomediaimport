package main

import (
	"fmt"
	"os/exec"
)

// sendNotification sends a macOS notification using osascript.
// It is fire-and-forget — it does not wait for delivery or check errors.
func sendNotification(title, message string) {
	script := fmt.Sprintf(`display notification %q with title %q`, message, title)
	cmd := exec.Command("osascript", "-e", script)
	_ = cmd.Start() // fire and forget
}
