package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Version = "0.3.2"

var rootCmd = &cobra.Command{
	Use:     "zuko",
	Short:   "Read-only CLI sandbox for AI agents",
	Long:    "Zuko wraps CLI tools (gh, himalaya, etc.) and enforces a read-only allowlist so AI agents can only run non-destructive commands.",
	Version: Version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
