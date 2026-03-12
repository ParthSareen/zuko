package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ParthSareen/zuko/auth"
	"github.com/spf13/cobra"
)

func init() {
	teardownOpenclawCmd.Flags().StringVar(&teardownOCPath, "config", "", "path to openclaw.json (default ~/.openclaw/openclaw.json)")
	teardownCmd.AddCommand(teardownOpenclawCmd)
}

var teardownOCPath string

var teardownOpenclawCmd = &cobra.Command{
	Use:   "openclaw",
	Short: "Remove zuko settings from openclaw.json",
	RunE:  runTeardownOpenclaw,
}

func runTeardownOpenclaw(_ *cobra.Command, _ []string) error {
	if err := auth.PromptAndVerifyPassword(); err != nil {
		return err
	}

	initOCPath = teardownOCPath
	return removeOpenclawSettings()
}

func removeOpenclawSettings() error {
	ocPath := resolveOpenclawPath()

	data, err := os.ReadFile(ocPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("openclaw config not found at %s", ocPath)
		}
		return fmt.Errorf("could not read %s: %w", ocPath, err)
	}

	var ocConfig map[string]any
	if err := json.Unmarshal(data, &ocConfig); err != nil {
		return fmt.Errorf("invalid JSON in %s: %w", ocPath, err)
	}

	changed := false

	if envMap, ok := ocConfig["env"].(map[string]any); ok {
		if _, ok := envMap["PATH"]; ok {
			delete(envMap, "PATH")
			changed = true
			if len(envMap) == 0 {
				delete(ocConfig, "env")
			}
		}
	}

	if toolsMap, ok := ocConfig["tools"].(map[string]any); ok {
		if execMap, ok := toolsMap["exec"].(map[string]any); ok {
			delete(execMap, "security")
			delete(execMap, "allowlist")
			changed = true
			if len(execMap) == 0 {
				delete(toolsMap, "exec")
			}
			if len(toolsMap) == 0 {
				delete(ocConfig, "tools")
			}
		}
	}

	if !changed {
		fmt.Printf("No zuko settings found in %s — nothing to remove.\n", ocPath)
		return nil
	}

	out, err := json.MarshalIndent(ocConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(ocPath, append(out, '\n'), 0600); err != nil {
		return fmt.Errorf("failed to write %s: %w", ocPath, err)
	}

	fmt.Printf("Removed zuko settings from %s\n", ocPath)
	return nil
}
