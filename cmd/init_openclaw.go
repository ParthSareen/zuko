package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ParthSareen/zuko/auth"
	"github.com/ParthSareen/zuko/config"
	"github.com/spf13/cobra"
)

func init() {
	initOpenclawCmd.Flags().BoolVar(&initOCDefenseInDepth, "defense-in-depth", false, "also add openclaw-level allowlist (both zuko + openclaw enforce)")
	initOpenclawCmd.Flags().StringVar(&initOCPath, "config", "", "path to openclaw.json (default ~/.openclaw/openclaw.json)")
	initCmd.AddCommand(initOpenclawCmd)
}

var (
	initOCDefenseInDepth bool
	initOCPath           string
)

var initOpenclawCmd = &cobra.Command{
	Use:   "openclaw",
	Short: "Merge zuko settings into openclaw.json (requires authentication)",
	RunE:  runInitOpenclaw,
}

func resolveOpenclawPath() string {
	if initOCPath != "" {
		return initOCPath
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".openclaw", "openclaw.json")
}

func runInitOpenclaw(_ *cobra.Command, _ []string) error {
	if err := auth.PromptAndVerifyPassword("init openclaw"); err != nil {
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

	ocPath := resolveOpenclawPath()

	existing, err := os.ReadFile(ocPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("openclaw config not found at %s — use --config to specify the path", ocPath)
		}
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

	if initOCDefenseInDepth {
		execMap["security"] = "allowlist"
		allowlist := buildAllowlist(cfg)
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
	if initOCDefenseInDepth {
		fmt.Println("  tools.exec.security → allowlist (defense in depth)")
	} else {
		fmt.Println("  tools.exec.security → full (zuko enforces allowlist)")
	}
	return nil
}
