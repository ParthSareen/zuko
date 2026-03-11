package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ParthSareen/zuko/auth"
	"github.com/ParthSareen/zuko/config"
	"github.com/spf13/cobra"
)

func init() {
	initCmd.Flags().BoolVar(&initDefenseInDepth, "defense-in-depth", false, "also add openclaw-level allowlist (both zuko + openclaw enforce)")
	rootCmd.AddCommand(initCmd)
}

var initDefenseInDepth bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Merge zuko settings into existing openclaw.json (requires authentication)",
	RunE:  runInit,
}

func openclawConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".openclaw", "openclaw.json")
}

func runInit(_ *cobra.Command, _ []string) error {
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

	ocPath := openclawConfigPath()
	existing, err := os.ReadFile(ocPath)
	if err != nil {
		return fmt.Errorf("could not read %s: %w", ocPath, err)
	}

	var ocConfig map[string]any
	if err := json.Unmarshal(existing, &ocConfig); err != nil {
		return fmt.Errorf("invalid JSON in %s: %w", ocPath, err)
	}

	// Prepend shim dir to PATH so zuko shims shadow the real binaries
	// but all other tools (git, node, python, etc.) remain accessible.
	envMap := getOrCreateMap(ocConfig, "env")
	envMap["PATH"] = shimDir + ":${PATH}"
	ocConfig["env"] = envMap

	// Merge tools.exec settings
	toolsMap := getOrCreateMap(ocConfig, "tools")
	execMap := getOrCreateMap(toolsMap, "exec")

	if initDefenseInDepth {
		execMap["security"] = "allowlist"
		allowlist := buildAllowlist(cfg)
		// Merge with any existing allowlist entries
		if existing, ok := execMap["allowlist"]; ok {
			if existingList, ok := existing.([]any); ok {
				seen := make(map[string]bool)
				for _, entry := range allowlist {
					seen[entry] = true
				}
				for _, entry := range existingList {
					if s, ok := entry.(string); ok && !seen[s] {
						allowlist = append(allowlist, s)
					}
				}
			}
		}
		execMap["allowlist"] = allowlist
	} else {
		execMap["security"] = "full"
	}

	toolsMap["exec"] = execMap
	ocConfig["tools"] = toolsMap

	data, err := json.MarshalIndent(ocConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(ocPath, append(data, '\n'), 0600); err != nil {
		return fmt.Errorf("failed to write %s: %w", ocPath, err)
	}

	fmt.Printf("Updated %s\n", ocPath)
	fmt.Printf("  env.PATH → %s:${PATH}\n", shimDir)
	if initDefenseInDepth {
		fmt.Println("  tools.exec.security → allowlist (defense in depth)")
	} else {
		fmt.Println("  tools.exec.security → full (zuko enforces allowlist)")
	}
	return nil
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
