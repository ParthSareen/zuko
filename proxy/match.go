package proxy

import (
	"path/filepath"
	"strings"

	"github.com/ParthSareen/zuko/config"
)

// Check returns whether the given args are allowed for the tool.
// It returns the matched subcommand name (for deny_flags lookup) and whether it's allowed.
func Check(tool config.Tool, args []string) (allowed bool, matched string) {
	if len(args) == 0 {
		return tool.AllowBare, ""
	}

	subcmds := extractSubcommands(args)
	if len(subcmds) == 0 {
		return tool.AllowBare, ""
	}

	for _, prefix := range tool.Allow {
		if matchesPrefix(subcmds, prefix) {
			// Check deny_flags for the matched prefix (specific override beats general)
			if len(prefix) > 0 {
				key := strings.Join(prefix, " ")
				if denied, _ := hasDeniedFlag(tool.DenyFlags, key, args); denied {
					return false, key
				}
			}
			return true, strings.Join(prefix, " ")
		}
	}
	return false, strings.Join(subcmds, " ")
}

// extractSubcommands pulls out positional (non-flag) tokens from args.
// Flags are tokens starting with "-". If a flag doesn't contain "=",
// the next token is assumed to be its value and skipped.
// Paths are normalized to their basenames to prevent bypasses like /usr/bin/git.
func extractSubcommands(args []string) []string {
	var subcmds []string
	skip := false
	for _, arg := range args {
		if skip {
			skip = false
			continue
		}
		if strings.HasPrefix(arg, "-") {
			// If flag doesn't use "=" form, skip next arg as its value
			if !strings.Contains(arg, "=") {
				skip = true
			}
			continue
		}
		// Normalize paths to basename (e.g., /usr/bin/git -> git)
		subcmds = append(subcmds, filepath.Base(arg))
	}
	return subcmds
}

// matchesPrefix checks if subcmds starts with the given prefix.
func matchesPrefix(subcmds, prefix []string) bool {
	if len(subcmds) < len(prefix) {
		return false
	}
	for i, p := range prefix {
		if subcmds[i] != p {
			return false
		}
	}
	return true
}

// CheckLocked returns whether the args match a locked subcommand for the tool.
// If matched, it returns true and the matched subcommand string (e.g. "commit").
func CheckLocked(tool config.Tool, args []string) (isLocked bool, subcmd string) {
	if len(tool.Locked) == 0 || len(args) == 0 {
		return false, ""
	}

	subcmds := extractSubcommands(args)
	if len(subcmds) == 0 {
		return false, ""
	}

	for _, prefix := range tool.Locked {
		if matchesPrefix(subcmds, prefix) {
			return true, strings.Join(prefix, " ")
		}
	}
	return false, ""
}

// hasDeniedFlag checks if any args contain a denied flag for the given subcommand key.
func hasDeniedFlag(denyFlags map[string][]string, key string, args []string) (bool, string) {
	denied, ok := denyFlags[key]
	if !ok {
		return false, ""
	}
	for _, arg := range args {
		for _, flag := range denied {
			if arg == flag || strings.HasPrefix(arg, flag+"=") || strings.HasPrefix(arg, flag) && len(arg) > len(flag) && arg[len(flag)] != '-' {
				return true, flag
			}
		}
	}
	return false, ""
}
