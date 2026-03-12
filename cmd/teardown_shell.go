package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/ParthSareen/zuko/auth"
	"github.com/spf13/cobra"
)

func init() {
	teardownShellCmd.Flags().StringVar(&teardownShellRC, "rc", "", "shell rc file to modify (default: auto-detect)")
	teardownCmd.AddCommand(teardownShellCmd)
}

var teardownShellRC string

var teardownShellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Remove zuko PATH block from shell rc file",
	RunE:  runTeardownShell,
}

func runTeardownShell(_ *cobra.Command, _ []string) error {
	if err := auth.PromptAndVerifyPassword(); err != nil {
		return err
	}

	initShellRC = teardownShellRC
	rcPath, err := resolveShellRC()
	if err != nil {
		return err
	}
	return removeShellBlock(rcPath)
}

func removeShellBlock(rcPath string) error {
	data, err := os.ReadFile(rcPath)
	if err != nil {
		return fmt.Errorf("could not read %s: %w", rcPath, err)
	}

	content := string(data)
	beginIdx := strings.Index(content, zukoMarkerBegin)
	if beginIdx == -1 {
		fmt.Printf("No zuko block found in %s — nothing to remove.\n", rcPath)
		return nil
	}

	endIdx := strings.Index(content, zukoMarkerEnd)
	if endIdx == -1 {
		return fmt.Errorf("found %s but no matching %s in %s — edit manually", zukoMarkerBegin, zukoMarkerEnd, rcPath)
	}
	endIdx += len(zukoMarkerEnd)
	if endIdx < len(content) && content[endIdx] == '\n' {
		endIdx++
	}
	if beginIdx > 0 && content[beginIdx-1] == '\n' {
		beginIdx--
	}

	content = content[:beginIdx] + content[endIdx:]

	if err := os.WriteFile(rcPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", rcPath, err)
	}

	fmt.Printf("Removed zuko block from %s\n", rcPath)
	return nil
}
