package proxy

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/ParthSareen/zuko/auth"
	"github.com/ParthSareen/zuko/config"
)

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
