package cmd

import (
	"fmt"
	"strconv"

	"github.com/ParthSareen/zuko/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(timeoutCmd)
}

var timeoutCmd = &cobra.Command{
	Use:   "timeout [minutes]",
	Short: "Get or set the default unlock duration in minutes",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runTimeout,
}

func runTimeout(_ *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if len(args) == 0 {
		mins := cfg.TimeoutMinutes
		if mins == 0 {
			mins = 5
		}
		fmt.Printf("%dm\n", mins)
		return nil
	}

	mins, err := strconv.Atoi(args[0])
	if err != nil || mins < 1 {
		return fmt.Errorf("timeout must be a positive integer (minutes)")
	}

	cfg.TimeoutMinutes = mins
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	fmt.Printf("Default unlock timeout set to %dm.\n", mins)
	return nil
}
