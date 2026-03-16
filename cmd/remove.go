package cmd

import (
	"fmt"

	"github.com/ParthSareen/zuko/auth"
	"github.com/ParthSareen/zuko/config"
	"github.com/ParthSareen/zuko/shim"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(removeCmd)
}

var removeCmd = &cobra.Command{
	Use:   "remove <tool>",
	Short: "Remove a CLI tool from zuko",
	Args:  cobra.ExactArgs(1),
	RunE:  runRemove,
}

func runRemove(_ *cobra.Command, args []string) error {
	if err := auth.PromptAndVerifyPassword("remove tool"); err != nil {
		return err
	}

	toolName := args[0]

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if _, exists := cfg.Tools[toolName]; !exists {
		return fmt.Errorf("%s is not configured", toolName)
	}

	delete(cfg.Tools, toolName)

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	if err := shim.Remove(cfg.ShimDir, toolName); err != nil {
		fmt.Printf("warning: could not remove shim: %v\n", err)
	}

	fmt.Printf("removed %s\n", toolName)
	return nil
}
