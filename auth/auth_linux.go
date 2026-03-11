package auth

import (
	"fmt"
	"os"
	"os/exec"
)

// PromptAndVerifyPassword validates the user's password via sudo.
// Runs `sudo -kv` which forces a fresh password prompt and validates
// against PAM without executing anything as root.
func PromptAndVerifyPassword() error {
	cmd := exec.Command("sudo", "-kv", "-p", "Zuko: authenticate to proceed\nPassword: ")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("authentication failed")
	}
	return nil
}
