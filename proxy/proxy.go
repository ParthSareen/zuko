package proxy

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/ParthSareen/zuko/auth"
	"github.com/ParthSareen/zuko/config"
)

func CopyToClipboard(text string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		cmd = exec.Command("xclip", "-selection", "clipboard")
	default:
		return
	}
	cmd.Stdin = strings.NewReader(text)
	cmd.Run()
}

func lastBlockedPath() string {
	return filepath.Join(config.ConfigDir(), "last-blocked")
}

func saveLastBlocked(cmd string) {
	os.WriteFile(lastBlockedPath(), []byte(cmd), 0600)
}

// LoadAndClearLastBlocked returns the last blocked command and removes the file.
func LoadAndClearLastBlocked() string {
	path := lastBlockedPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	os.Remove(path)
	return string(data)
}

func Run(toolName string, args []string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "zuko: %v\n", err)
		os.Exit(1)
	}

	tool, ok := cfg.Tools[toolName]
	if !ok {
		fmt.Fprintf(os.Stderr, "zuko: unknown tool %q — not configured\n", toolName)
		os.Exit(1)
	}

	if tool.RealBinary == "" {
		fmt.Fprintf(os.Stderr, "zuko: no real_binary configured for %q — run 'zuko setup'\n", toolName)
		os.Exit(1)
	}

	// If unlocked, skip allowlist enforcement
	if auth.IsUnlocked() {
		argv := append([]string{toolName}, args...)
		if err := syscall.Exec(tool.RealBinary, argv, os.Environ()); err != nil {
			fmt.Fprintf(os.Stderr, "zuko: exec %s: %v\n", tool.RealBinary, err)
			os.Exit(127)
		}
	}

	// Check locked subcommands before the allowlist
	if isLocked, subcmd := CheckLocked(tool, args); isLocked {
		scope := toolName + ":" + subcmd
		if auth.IsGranted(scope) {
			argv := append([]string{toolName}, args...)
			if err := syscall.Exec(tool.RealBinary, argv, os.Environ()); err != nil {
				fmt.Fprintf(os.Stderr, "zuko: exec %s: %v\n", tool.RealBinary, err)
				os.Exit(127)
			}
		}
		unlockCmd := fmt.Sprintf("zuko unlock %s %s", toolName, subcmd)
		originalCmd := toolName + " " + strings.Join(args, " ")
		CopyToClipboard(unlockCmd)
		saveLastBlocked(originalCmd)
		fmt.Fprintf(os.Stderr, "zuko: %s %s requires unlock — run '%s' (copied to clipboard)\n",
			toolName, subcmd, unlockCmd)
		os.Exit(1)
	}

	// AllowAll: everything not locked passes through
	if tool.AllowAll {
		argv := append([]string{toolName}, args...)
		if err := syscall.Exec(tool.RealBinary, argv, os.Environ()); err != nil {
			fmt.Fprintf(os.Stderr, "zuko: exec %s: %v\n", tool.RealBinary, err)
			os.Exit(127)
		}
	}

	allowed, matched := Check(tool, args)
	if !allowed {
		cmd := toolName
		if matched != "" {
			cmd = toolName + " " + matched
		} else if len(args) > 0 {
			cmd = toolName + " " + strings.Join(args, " ")
		}
		fmt.Fprintf(os.Stderr, "zuko: blocked %q — not in allowlist\n", cmd)
		os.Exit(1)
	}

	argv := append([]string{toolName}, args...)
	if err := syscall.Exec(tool.RealBinary, argv, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "zuko: exec %s: %v\n", tool.RealBinary, err)
		os.Exit(127)
	}
}
