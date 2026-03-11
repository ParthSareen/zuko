package cmd

import (
	"fmt"
	"time"

	"github.com/ParthSareen/zuko/auth"
	"github.com/spf13/cobra"
)

func init() {
	unlockCmd.Flags().DurationVarP(&unlockDuration, "duration", "d", auth.DefaultUnlockDuration, "unlock duration")
	rootCmd.AddCommand(unlockCmd)
}

var unlockDuration time.Duration

var unlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Authenticate and temporarily allow all commands through shims",
	RunE:  runUnlock,
}

func runUnlock(_ *cobra.Command, _ []string) error {
	if err := auth.PromptAndVerifyPassword(); err != nil {
		return err
	}

	if err := auth.Unlock(unlockDuration); err != nil {
		return fmt.Errorf("failed to unlock: %w", err)
	}

	fmt.Printf("Unlocked for %s. Run 'zuko lock' to re-lock.\n", unlockDuration)
	return nil
}
