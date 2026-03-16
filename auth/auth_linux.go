package auth

import (
	"fmt"
	"os"
	"os/exec"
)

// PromptAndVerifyPassword validates the user's password via sudo.
func PromptAndVerifyPassword(reason string) error {
	if reason == "" {
		reason = "authenticate to proceed"
	}
	prompt := fmt.Sprintf("Zuko: %s\nPassword: ", reason)

	cmd := exec.Command("sudo", "-kv", "-p", prompt)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("authentication failed")
	}
	return nil
}
