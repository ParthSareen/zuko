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

// hasUnlockedFlag checks if any unlocked flags are present for the given subcommand.
// Unlocked flags bypass the lock for that command.
func hasUnlockedFlag(unlockedFlags map[string][]string, subcmd string, args []string) bool {
	if unlockedFlags == nil {
		return false
	}
	flags, ok := unlockedFlags[subcmd]
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

	// If globally unlocked, skip all checks
	if auth.IsUnlocked() {
		log.Write(log.Entry{Tool: toolName, Args: args, Action: "allowed"})
		execTool(toolName, tool.RealBinary, args)
		return
	}

	// Check allowlist first - takes precedence over locked
	allowed, matched := Check(tool, args)
	if allowed {
		// Check for deny flags
		if denied, flag := hasDeniedFlag(tool.DenyFlags, matched, args); denied {
			log.Write(log.Entry{Tool: toolName, Args: args, Action: "blocked", Scope: matched, Error: "denied flag: " + flag})
			fmt.Fprintf(os.Stderr, "zuko: %s %s with flag %q is blocked\n", toolName, matched, flag)
			os.Exit(1)
		}
		log.Write(log.Entry{Tool: toolName, Args: args, Action: "allowed"})
		execTool(toolName, tool.RealBinary, args)
		return
	}

	// Check bare command (no args)
	if len(args) == 0 {
		if tool.AllowBare {
			log.Write(log.Entry{Tool: toolName, Args: args, Action: "allowed"})
			execTool(toolName, tool.RealBinary, args)
			return
		}
		// A tool-level unlock permits bare invocation.
		if auth.IsGranted(toolName) {
			log.Write(log.Entry{Tool: toolName, Args: args, Action: "allowed", Scope: toolName})
			execTool(toolName, tool.RealBinary, args)
			return
		}
		log.Write(log.Entry{Tool: toolName, Args: args, Action: "blocked", Scope: "bare"})
		unlockCmd := fmt.Sprintf("zuko unlock %s", toolName)
		CopyToClipboard(unlockCmd)
		saveLastBlocked(toolName)
		fmt.Fprintf(os.Stderr, "zuko: %s (bare) is not allowed — run '%s' to allow (copied to clipboard)\n",
			toolName, unlockCmd)
		os.Exit(1)
	}

	// AllowAll: everything passes through (except locked with dangerous flags)
	if tool.AllowAll {
		// Check if this is a locked command with dangerous flags
		if isLocked, subcmd := CheckLocked(tool, args); isLocked {
			scope := toolName + ":" + subcmd
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
			// Check if unlocked flags are present - bypass lock
			if hasUnlockedFlag(tool.UnlockedFlags, subcmd, args) {
				log.Write(log.Entry{Tool: toolName, Args: args, Action: "allowed", Scope: scope})
				execTool(toolName, tool.RealBinary, args)
				return
			}
			// Locked but not dangerous - require auth
			if auth.IsGranted(scope) {
				log.Write(log.Entry{Tool: toolName, Args: args, Action: "allowed", Scope: scope})
				execTool(toolName, tool.RealBinary, args)
				return
			}
			log.Write(log.Entry{Tool: toolName, Args: args, Action: "blocked", Scope: scope})
			reason := toolName + " " + subcmd
			if err := auth.PromptAndVerifyPassword(reason); err != nil {
				log.Write(log.Entry{Tool: toolName, Args: args, Action: "auth_failed", Scope: scope, Error: err.Error()})
				unlockCmd := fmt.Sprintf("zuko unlock %s %s", toolName, subcmd)
				originalCmd := toolName + " " + strings.Join(args, " ")
				CopyToClipboard(unlockCmd)
				saveLastBlocked(originalCmd)
				fmt.Fprintf(os.Stderr, "zuko: %s %s requires unlock — run '%s' (copied to clipboard)\n",
					toolName, subcmd, unlockCmd)
				os.Exit(1)
			}
			log.Write(log.Entry{Tool: toolName, Args: args, Action: "granted", Scope: scope})
			if err := auth.UnlockScope(scope, resolveTimeout(cfg)); err != nil {
				fmt.Fprintf(os.Stderr, "zuko: failed to unlock: %v\n", err)
				os.Exit(1)
			}
			execTool(toolName, tool.RealBinary, args)
			return
		}
		// Not locked, allow through
		log.Write(log.Entry{Tool: toolName, Args: args, Action: "allowed"})
		execTool(toolName, tool.RealBinary, args)
		return
	}

	// Check locked subcommands (no AllowAll)
	if isLocked, subcmd := CheckLocked(tool, args); isLocked {
		scope := toolName + ":" + subcmd
		if auth.IsGranted(scope) {
			log.Write(log.Entry{Tool: toolName, Args: args, Action: "allowed", Scope: scope})
			execTool(toolName, tool.RealBinary, args)
			return
		}
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
		log.Write(log.Entry{Tool: toolName, Args: args, Action: "blocked", Scope: scope})
		reason := toolName + " " + subcmd
		if err := auth.PromptAndVerifyPassword(reason); err != nil {
			log.Write(log.Entry{Tool: toolName, Args: args, Action: "auth_failed", Scope: scope, Error: err.Error()})
			unlockCmd := fmt.Sprintf("zuko unlock %s %s", toolName, subcmd)
			originalCmd := toolName + " " + strings.Join(args, " ")
			CopyToClipboard(unlockCmd)
			saveLastBlocked(originalCmd)
			fmt.Fprintf(os.Stderr, "zuko: %s %s requires unlock — run '%s' (copied to clipboard)\n",
				toolName, subcmd, unlockCmd)
			os.Exit(1)
		}
		log.Write(log.Entry{Tool: toolName, Args: args, Action: "granted", Scope: scope})
		if err := auth.UnlockScope(scope, resolveTimeout(cfg)); err != nil {
			fmt.Fprintf(os.Stderr, "zuko: failed to unlock: %v\n", err)
			os.Exit(1)
		}
		execTool(toolName, tool.RealBinary, args)
		return
	}

	// Not in allowlist, not locked, not AllowAll.
	// Check if an unlock scope covers this command so that e.g.
	// `zuko unlock gh issue` permits `gh issue edit`.
	subcmds := extractSubcommands(args)
	if len(subcmds) > 0 {
		scope := toolName + ":" + strings.Join(subcmds, " ")
		if auth.IsGranted(scope) {
			log.Write(log.Entry{Tool: toolName, Args: args, Action: "allowed", Scope: scope})
			execTool(toolName, tool.RealBinary, args)
			return
		}
	}

	log.Write(log.Entry{Tool: toolName, Args: args, Action: "blocked", Scope: "allowlist"})
	if len(subcmds) > 0 {
		unlockCmd := fmt.Sprintf("zuko unlock %s %s", toolName, subcmds[0])
		originalCmd := toolName + " " + strings.Join(args, " ")
		CopyToClipboard(unlockCmd)
		saveLastBlocked(originalCmd)
		fmt.Fprintf(os.Stderr, "zuko: %s %s is not in allowlist — run '%s' to allow (copied to clipboard)\n",
			toolName, strings.Join(args, " "), unlockCmd)
	} else {
		fmt.Fprintf(os.Stderr, "zuko: %s %s is not in allowlist\n", toolName, strings.Join(args, " "))
	}
	os.Exit(1)
}
