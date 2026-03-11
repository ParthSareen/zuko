package auth

import (
	"fmt"
	"os"
	"os/exec"
)

// PromptAndVerifyPassword shows the native macOS authentication dialog.
// Uses osascript to trigger a system-level admin auth prompt.
func PromptAndVerifyPassword() error {
	cmd := exec.Command("osascript", "-e",
		`do shell script "echo ok" with administrator privileges with prompt "Zuko: authenticate to proceed"`)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("authentication failed")
	}
	return nil
}
