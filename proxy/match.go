package proxy

import (
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
			// Check deny_flags for the first token of the matched prefix
			if len(prefix) > 0 {
				if denied, _ := hasDeniedFlag(tool.DenyFlags, prefix[0], args); denied {
					return false, strings.Join(prefix, " ")
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
		subcmds = append(subcmds, arg)
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
