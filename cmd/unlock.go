package cmd

import (
	"fmt"
	"strings"
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
	Use:   "unlock [tool] [subcommand]",
	Short: "Authenticate and temporarily allow commands through shims",
	Args:  cobra.MaximumNArgs(2),
	RunE:  runUnlock,
}

func runUnlock(_ *cobra.Command, args []string) error {
	if err := auth.PromptAndVerifyPassword(); err != nil {
		return err
	}

	switch len(args) {
	case 0:
		if err := auth.Unlock(unlockDuration); err != nil {
			return fmt.Errorf("failed to unlock: %w", err)
		}
		fmt.Printf("Unlocked globally for %s. Run 'zuko lock' to re-lock.\n", unlockDuration)
	case 1:
		scope := args[0]
		if err := auth.UnlockScope(scope, unlockDuration); err != nil {
			return fmt.Errorf("failed to unlock %s: %w", scope, err)
		}
		fmt.Printf("Unlocked %s for %s. Run 'zuko lock %s' to re-lock.\n", scope, unlockDuration, scope)
	case 2:
		scope := args[0] + ":" + args[1]
		if err := auth.UnlockScope(scope, unlockDuration); err != nil {
			return fmt.Errorf("failed to unlock %s: %w", scope, err)
		}
		fmt.Printf("Unlocked %s for %s. Run 'zuko lock %s' to re-lock.\n",
			strings.Replace(scope, ":", " ", 1), unlockDuration, strings.Replace(scope, ":", " ", 1))
	}

	return nil
}
