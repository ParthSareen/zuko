package cmd

import (
	"fmt"
	"os"

	"github.com/ParthSareen/zuko/auth"
	"github.com/ParthSareen/zuko/config"
	"github.com/ParthSareen/zuko/shim"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(teardownCmd)
}

var teardownCmd = &cobra.Command{
	Use:   "teardown",
	Short: "Remove zuko shims and undo init changes",
	Long: `Remove zuko shims and optionally undo init changes.

Running bare 'zuko teardown' removes shim symlinks only.

Subcommands:
  shell      Remove the zuko PATH block from your shell rc file
  openclaw   Remove zuko settings from openclaw.json
  all        Remove shims and undo both shell and openclaw init`,
	RunE: runTeardown,
}

func runTeardown(_ *cobra.Command, _ []string) error {
	if err := auth.PromptAndVerifyPassword("teardown"); err != nil {
		return err
	}
	return removeShims()
}

func removeShims() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	for name := range cfg.Tools {
		if err := shim.Remove(cfg.ShimDir, name); err != nil {
			if !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "warning: could not remove shim for %s: %v\n", name, err)
			}
			continue
		}
		fmt.Printf("removed shim for %s\n", name)
	}
	return nil
}
