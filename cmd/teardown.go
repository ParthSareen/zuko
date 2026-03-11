package cmd

import (
	"fmt"
	"os"

	"github.com/ParthSareen/zuko/config"
	"github.com/ParthSareen/zuko/shim"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(teardownCmd)
}

var teardownCmd = &cobra.Command{
	Use:   "teardown",
	Short: "Remove shim symlinks",
	RunE:  runTeardown,
}

func runTeardown(_ *cobra.Command, _ []string) error {
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

	fmt.Println("\nShims removed. You can re-run 'zuko setup' to recreate them.")
	return nil
}
