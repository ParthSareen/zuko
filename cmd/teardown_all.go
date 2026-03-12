package cmd

import (
	"fmt"

	"github.com/ParthSareen/zuko/auth"
	"github.com/spf13/cobra"
)

func init() {
	teardownCmd.AddCommand(teardownAllCmd)
}

var teardownAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Remove shims and undo both shell and openclaw init",
	RunE:  runTeardownAll,
}

func runTeardownAll(cmd *cobra.Command, args []string) error {
	if err := auth.PromptAndVerifyPassword(); err != nil {
		return err
	}

	fmt.Println("Removing shims...")
	if err := removeShims(); err != nil {
		fmt.Printf("warning: %v\n", err)
	}

	fmt.Println("\nRemoving shell rc block...")
	// Skip auth for sub-calls since we already authenticated
	rcPath, err := resolveShellRC()
	if err != nil {
		fmt.Printf("skipping shell: %v\n", err)
	} else {
		teardownShellRC = ""
		initShellRC = ""
		if err := removeShellBlock(rcPath); err != nil {
			fmt.Printf("warning: %v\n", err)
		}
	}

	fmt.Println("\nRemoving openclaw settings...")
	initOCPath = ""
	teardownOCPath = ""
	if err := removeOpenclawSettings(); err != nil {
		fmt.Printf("skipping openclaw: %v\n", err)
	}

	fmt.Println("\nDone. Restart your shell to apply changes.")
	return nil
}
