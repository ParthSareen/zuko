package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ParthSareen/zuko/log"
	"github.com/spf13/cobra"
)

func init() {
	logsCmd.Flags().IntVarP(&logsLimit, "limit", "n", 50, "number of entries to show")
	logsCmd.Flags().BoolVarP(&logsClear, "clear", "c", false, "clear all logs")
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "follow log output")
	logsCmd.Flags().BoolVarP(&logsRotate, "rotate", "r", false, "keep only last 1000 entries")
	rootCmd.AddCommand(logsCmd)
}

var logsLimit int
var logsClear bool
var logsFollow bool
var logsRotate bool

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

	if logsRotate {
		if err := log.Rotate(1000); err != nil {
			return fmt.Errorf("failed to rotate logs: %w", err)
		}
		fmt.Println("Logs rotated (kept last 1000 entries).")
		return nil
	}

	if logsFollow {
		return followLogs()
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

func followLogs() error {
	logPath := log.LogsPath()

	// Open file
	f, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No logs found.")
			return nil
		}
		return fmt.Errorf("failed to open logs: %w", err)
	}
	defer f.Close()

	// Seek to end
	f.Seek(0, io.SeekEnd)

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Ticker for polling
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	fmt.Println("Following logs... (Ctrl+C to stop)")

	reader := bufio.NewReader(f)
	for {
		select {
		case <-sigCh:
			fmt.Println()
			return nil
		case <-ticker.C:
			for {
				line, err := reader.ReadBytes('\n')
				if err != nil {
					break
				}
				if len(line) == 0 {
					continue
				}
				var entry log.Entry
				if err := json.Unmarshal(line, &entry); err != nil {
					continue
				}
				fmt.Println(log.FormatEntry(entry))
			}
		}
	}
}

func init() {
	// Add shell completion for -n flag
	logsCmd.RegisterFlagCompletionFunc("limit", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{"10", "50", "100", "500"}, cobra.ShellCompDirectiveNoFileComp
	})
}
