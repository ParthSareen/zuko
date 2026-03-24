package cmd

import (
	"fmt"
	"os"

	"github.com/ParthSareen/zuko/log"
	"github.com/spf13/cobra"
)

func init() {
	logsCmd.Flags().IntVarP(&logsLimit, "limit", "n", 50, "number of entries to show")
	logsCmd.Flags().BoolVarP(&logsClear, "clear", "c", false, "clear all logs")
	rootCmd.AddCommand(logsCmd)
}

var logsLimit int
var logsClear bool

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View access logs",
	Long:  "View recent access logs showing blocked, granted, and allowed commands.",
	RunE:  runLogs,
}

func runLogs(_ *cobra.Command, _ []string) error {
	if logsClear {
		if err := log.Clear(); err != nil {
			return fmt.Errorf("failed to clear logs: %w", err)
		}
		fmt.Println("Logs cleared.")
		return nil
	}

	entries, err := log.Read(logsLimit)
	if err != nil {
		return fmt.Errorf("failed to read logs: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No logs found.")
		return nil
	}

	for _, e := range entries {
		fmt.Println(log.FormatEntry(e))
	}

	// Show count if truncated
	if logsLimit > 0 && len(entries) == logsLimit {
		fmt.Fprintf(os.Stderr, "\n(showing last %d entries; use -n for more)\n", logsLimit)
	}

	return nil
}

func init() {
	// Add shell completion for -n flag
	logsCmd.RegisterFlagCompletionFunc("limit", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{"10", "50", "100", "500"}, cobra.ShellCompDirectiveNoFileComp
	})
}
