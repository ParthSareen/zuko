package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ParthSareen/zuko/auth"
	"github.com/ParthSareen/zuko/config"
	"github.com/spf13/cobra"
)

func init() {
	initShellCmd.Flags().StringVar(&initShellRC, "rc", "", "shell rc file to modify (default: auto-detect ~/.zshrc or ~/.bashrc)")
	initCmd.AddCommand(initShellCmd)
}

var initShellRC string

var initShellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Prepend zuko shim dir to PATH in your shell rc file",
	RunE:  runInitShell,
}

const zukoMarkerBegin = "# >>> zuko >>>"
const zukoMarkerEnd = "# <<< zuko <<<"

func resolveShellRC() (string, error) {
	if initShellRC != "" {
		return initShellRC, nil
	}
	home, _ := os.UserHomeDir()
	shell := os.Getenv("SHELL")
	switch {
	case strings.HasSuffix(shell, "/zsh"):
		return filepath.Join(home, ".zshrc"), nil
	case strings.HasSuffix(shell, "/bash"):
		return filepath.Join(home, ".bashrc"), nil
	default:
		return "", fmt.Errorf("could not detect shell rc file for %s — use --rc to specify", shell)
	}
}

func runInitShell(_ *cobra.Command, _ []string) error {
	if err := auth.PromptAndVerifyPassword(); err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("run 'zuko setup' first: %w", err)
	}

	shimDir := cfg.ShimDir
	if shimDir == "" {
		home, _ := os.UserHomeDir()
		shimDir = filepath.Join(home, ".config", "zuko", "shims")
	}

	rcPath, err := resolveShellRC()
	if err != nil {
		return err
	}

	snippet := fmt.Sprintf(`%s
export PATH="%s:$PATH"
%s`, zukoMarkerBegin, shimDir, zukoMarkerEnd)

	existing, err := os.ReadFile(rcPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("could not read %s: %w", rcPath, err)
	}

	content := string(existing)

	// Replace existing zuko block if present
	if beginIdx := strings.Index(content, zukoMarkerBegin); beginIdx != -1 {
		if endIdx := strings.Index(content, zukoMarkerEnd); endIdx != -1 {
			endIdx += len(zukoMarkerEnd)
			// Trim trailing newline after end marker
			if endIdx < len(content) && content[endIdx] == '\n' {
				endIdx++
			}
			content = content[:beginIdx] + snippet + "\n" + content[endIdx:]
		}
	} else {
		// Append
		if len(content) > 0 && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += "\n" + snippet + "\n"
	}

	if err := os.WriteFile(rcPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", rcPath, err)
	}

	fmt.Printf("Updated %s\n", rcPath)
	fmt.Printf("  PATH prepended with %s\n", shimDir)
	fmt.Println("\nRestart your shell or run:")
	fmt.Printf("  source %s\n", rcPath)
	return nil
}
