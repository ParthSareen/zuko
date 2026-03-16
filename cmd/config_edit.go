package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/ParthSareen/zuko/auth"
	"github.com/ParthSareen/zuko/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(configCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Edit the zuko config (requires authentication)",
	RunE:  runConfigEdit,
}

func runConfigEdit(_ *cobra.Command, _ []string) error {
	if err := auth.PromptAndVerifyPassword("edit config"); err != nil {
		return err
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	path := config.ConfigPath()
	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor exited with error: %w", err)
	}

	fmt.Println("Config updated.")
	return nil
}
