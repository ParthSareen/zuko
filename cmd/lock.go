package cmd

import (
	"fmt"

	"github.com/ParthSareen/zuko/auth"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(lockCmd)
}

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Re-lock shims immediately (revoke unlock session)",
	RunE:  runLock,
}

func runLock(_ *cobra.Command, _ []string) error {
	if err := auth.Lock(); err != nil {
		return fmt.Errorf("failed to lock: %w", err)
	}
	fmt.Println("Locked.")
	return nil
}
