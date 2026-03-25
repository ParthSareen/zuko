package proxy

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/ParthSareen/zuko/auth"
	"github.com/ParthSareen/zuko/config"
	"github.com/ParthSareen/zuko/log"
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

func execTool(toolName string, realBinary string, args []string) {
	argv := append([]string{toolName}, args...)
	if err := syscall.Exec(realBinary, argv, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "zuko: exec %s: %v\n", realBinary, err)
		os.Exit(127)
	}
}

func resolveTimeout(cfg *config.Config) time.Duration {
	if cfg.TimeoutMinutes > 0 {
		return time.Duration(cfg.TimeoutMinutes) * time.Minute
	}
	return auth.DefaultUnlockDuration
}

// hasDangerousFlag checks if any dangerous flags are present for the given subcommand.
// Dangerous flags trigger clipboard-only mode (no auto-prompt).
func hasDangerousFlag(dangerousFlags map[string][]string, subcmd string, args []string) bool {
	if dangerousFlags == nil {
		return false
	}
	flags, ok := dangerousFlags[subcmd]
	if !ok {
		return false
	}
	for _, arg := range args {
		for _, flag := range flags {
			if arg == flag || strings.HasPrefix(arg, flag+"=") {
				return true
			}
		}
	}
	return false
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

	// Check locked subcommands first (takes priority over AllowAll)
	if isLocked, subcmd := CheckLocked(tool, args); isLocked {
		scope := toolName + ":" + subcmd
		// Check if globally unlocked or specifically granted for this scope
		if auth.IsUnlocked() || auth.IsGranted(scope) {
			log.Write(log.Entry{Tool: toolName, Args: args, Action: "allowed", Scope: scope})
			execTool(toolName, tool.RealBinary, args)
			return
		}

		// Check for dangerous flags that trigger clipboard-only mode
		if hasDangerousFlag(tool.DangerousFlags, subcmd, args) {
			log.Write(log.Entry{Tool: toolName, Args: args, Action: "blocked_dangerous", Scope: scope})
			unlockCmd := fmt.Sprintf("zuko unlock %s %s", toolName, subcmd)
			originalCmd := toolName + " " + strings.Join(args, " ")
			CopyToClipboard(unlockCmd)
			saveLastBlocked(originalCmd)
			fmt.Fprintf(os.Stderr, "zuko: %s %s requires unlock — run '%s' (copied to clipboard)\n",
				toolName, subcmd, unlockCmd)
			os.Exit(1)
		}

		// Prompt for auth inline (tier 1: mostly safe commands)
		log.Write(log.Entry{Tool: toolName, Args: args, Action: "blocked", Scope: scope})
		reason := toolName + " " + subcmd
		if err := auth.PromptAndVerifyPassword(reason); err != nil {
			// Auth failed/cancelled — fall back to clipboard method
			log.Write(log.Entry{Tool: toolName, Args: args, Action: "auth_failed", Scope: scope, Error: err.Error()})
			unlockCmd := fmt.Sprintf("zuko unlock %s %s", toolName, subcmd)
			originalCmd := toolName + " " + strings.Join(args, " ")
			CopyToClipboard(unlockCmd)
			saveLastBlocked(originalCmd)
			fmt.Fprintf(os.Stderr, "zuko: %s %s requires unlock — run '%s' (copied to clipboard)\n",
				toolName, subcmd, unlockCmd)
			os.Exit(1)
		}

		// Auth succeeded — grant scoped unlock and execute
		log.Write(log.Entry{Tool: toolName, Args: args, Action: "granted", Scope: scope})
		if err := auth.UnlockScope(scope, resolveTimeout(cfg)); err != nil {
			fmt.Fprintf(os.Stderr, "zuko: failed to unlock: %v\n", err)
			os.Exit(1)
		}
		execTool(toolName, tool.RealBinary, args)
		return
	}

	// If globally unlocked, skip allowlist enforcement
	if auth.IsUnlocked() {
		log.Write(log.Entry{Tool: toolName, Args: args, Action: "allowed"})
		execTool(toolName, tool.RealBinary, args)
		return
	}

	// AllowAll: everything not locked passes through
	if tool.AllowAll {
		log.Write(log.Entry{Tool: toolName, Args: args, Action: "allowed"})
		execTool(toolName, tool.RealBinary, args)
		return
	}

	allowed, matched := Check(tool, args)
	if !allowed {
		// Prompt for auth inline for allowlist blocks
		log.Write(log.Entry{Tool: toolName, Args: args, Action: "blocked", Scope: "allowlist"})
		reason := "unlock all commands"
		if err := auth.PromptAndVerifyPassword(reason); err != nil {
			// Auth failed/cancelled — fall back to showing the block message
			log.Write(log.Entry{Tool: toolName, Args: args, Action: "auth_failed", Error: err.Error()})
			cmd := toolName
			if matched != "" {
				cmd = toolName + " " + matched
			} else if len(args) > 0 {
				cmd = toolName + " " + strings.Join(args, " ")
			}
			fmt.Fprintf(os.Stderr, "zuko: blocked %q — not in allowlist\n", cmd)
			os.Exit(1)
		}

		// Auth succeeded — grant global unlock and execute
		log.Write(log.Entry{Tool: toolName, Args: args, Action: "granted"})
		if err := auth.Unlock(resolveTimeout(cfg)); err != nil {
			fmt.Fprintf(os.Stderr, "zuko: failed to unlock: %v\n", err)
			os.Exit(1)
		}
		execTool(toolName, tool.RealBinary, args)
		return
	}

	log.Write(log.Entry{Tool: toolName, Args: args, Action: "allowed"})
	execTool(toolName, tool.RealBinary, args)
}
