package cmd

import (
	"fmt"
	"os"

	"github.com/ParthSareen/zuko/config"
	"github.com/ParthSareen/zuko/shim"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(setupCmd)
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Discover binaries and create shim symlinks",
	RunE:  runSetup,
}

func runSetup(_ *cobra.Command, _ []string) error {
	// Load existing config or create default
	cfg, err := config.Load()
	if err != nil {
		fmt.Println("No existing config found, creating defaults...")
		cfg = config.DefaultConfig()
		cfg.ExpandPaths()
	}

	// Merge in any new default tools that aren't in the existing config
	defaults := config.DefaultConfig()
	for name, tool := range defaults.Tools {
		if _, exists := cfg.Tools[name]; !exists {
			cfg.Tools[name] = tool
		}
	}

	shimDir := cfg.ShimDir
	if shimDir == "" {
		shimDir = config.ConfigDir() + "/shims"
		cfg.ShimDir = shimDir
	}

	zukoPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine zuko binary path: %w", err)
	}

	for name, tool := range cfg.Tools {
		// Discover real binary if not already set
		if tool.RealBinary == "" {
			path, err := shim.DiscoverBinary(name, shimDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: %v — skipping %s\n", err, name)
				continue
			}
			tool.RealBinary = path
			cfg.Tools[name] = tool
			fmt.Printf("discovered %s at %s\n", name, path)
		} else {
			fmt.Printf("using configured %s at %s\n", name, tool.RealBinary)
		}

		// Create shim
		if err := shim.Install(shimDir, zukoPath, name); err != nil {
			return fmt.Errorf("failed to create shim for %s: %w", name, err)
		}
		fmt.Printf("created shim %s/%s → %s\n", shimDir, name, zukoPath)
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	fmt.Printf("\nConfig saved to %s\n", config.ConfigPath())
	fmt.Printf("\nTo activate, set your agent's PATH:\n  export PATH=%s\n", shimDir)
	return nil
}
