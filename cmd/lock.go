package cmd

import (
	"fmt"
	"strings"

	"github.com/ParthSareen/zuko/auth"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(lockCmd)
}

var lockCmd = &cobra.Command{
	Use:   "lock [tool] [subcommand]",
	Short: "Re-lock shims immediately (revoke unlock session)",
	Args:  cobra.MaximumNArgs(2),
	RunE:  runLock,
}

func runLock(_ *cobra.Command, args []string) error {
	switch len(args) {
	case 0:
		if err := auth.Lock(); err != nil {
			return fmt.Errorf("failed to lock: %w", err)
		}
		fmt.Println("Locked all.")
	case 1:
		scope := args[0]
		if err := auth.LockScope(scope); err != nil {
			return fmt.Errorf("failed to lock %s: %w", scope, err)
		}
		fmt.Printf("Locked %s.\n", scope)
	case 2:
		scope := args[0] + ":" + args[1]
		if err := auth.LockScope(scope); err != nil {
			return fmt.Errorf("failed to lock %s: %w", scope, err)
		}
		fmt.Printf("Locked %s.\n", strings.Replace(scope, ":", " ", 1))
	}

	return nil
}
