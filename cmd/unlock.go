package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/ParthSareen/zuko/auth"
	"github.com/ParthSareen/zuko/config"
	"github.com/spf13/cobra"
)

func init() {
	unlockCmd.Flags().DurationVarP(&unlockDuration, "duration", "d", 0, "unlock duration (default: from 'zuko timeout' or 5m)")
	rootCmd.AddCommand(unlockCmd)
}

var unlockDuration time.Duration

var unlockCmd = &cobra.Command{
	Use:   "unlock [tool] [subcommand]",
	Short: "Authenticate and temporarily allow commands through shims",
	Args:  cobra.MaximumNArgs(2),
	RunE:  runUnlock,
}

func resolveTimeout() time.Duration {
	if unlockDuration != 0 {
		return unlockDuration
	}
	if cfg, err := config.Load(); err == nil && cfg.TimeoutMinutes > 0 {
		return time.Duration(cfg.TimeoutMinutes) * time.Minute
	}
	return auth.DefaultUnlockDuration
}

func runUnlock(_ *cobra.Command, args []string) error {
	if err := auth.PromptAndVerifyPassword(); err != nil {
		return err
	}

	unlockDuration = resolveTimeout()

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
