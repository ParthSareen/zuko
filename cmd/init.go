package cmd

import (
	"strings"

	"github.com/ParthSareen/zuko/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Configure PATH to use zuko shims",
	Long: `Configure PATH so CLI tools resolve to zuko shims.

Subcommands:
  shell      Prepend shim dir to PATH in your shell rc file (~/.zshrc, ~/.bashrc)
  openclaw   Merge zuko settings into ~/.openclaw/openclaw.json`,
}

// getOrCreateMap returns the nested map at key, creating it if missing or wrong type.
func getOrCreateMap(parent map[string]any, key string) map[string]any {
	if v, ok := parent[key]; ok {
		if m, ok := v.(map[string]any); ok {
			return m
		}
	}
	m := make(map[string]any)
	parent[key] = m
	return m
}

func buildAllowlist(cfg *config.Config) []string {
	var allowlist []string
	for name, tool := range cfg.Tools {
		for _, prefix := range tool.Allow {
			var b strings.Builder
			b.WriteString(name)
			for _, token := range prefix {
				b.WriteByte(' ')
				b.WriteString(token)
			}
			b.WriteString(" *")
			allowlist = append(allowlist, b.String())
		}
		if tool.AllowBare {
			allowlist = append(allowlist, name)
		}
	}
	return allowlist
}
