package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/ParthSareen/zuko/auth"
	"github.com/ParthSareen/zuko/config"
	"github.com/ParthSareen/zuko/shim"
	"github.com/spf13/cobra"
)

func init() {
	addCmd.Flags().StringSliceVar(&addAllow, "allow", nil, "allowed subcommand prefixes (e.g. get,describe,logs)")
	addCmd.Flags().BoolVar(&addPassthrough, "passthrough", false, "allow all commands (no filtering)")
	rootCmd.AddCommand(addCmd)
}

var (
	addAllow       []string
	addPassthrough bool
)

var addCmd = &cobra.Command{
	Use:   "add <tool> [flags]",
	Short: "Add a new CLI tool to zuko",
	Long: `Add a new CLI tool to the zuko sandbox.

Examples:
  zuko add jq --passthrough              # allow all jq commands
  zuko add kubectl --allow get,describe   # only allow get and describe
  zuko add docker --allow ps,images,logs  # only allow read-only docker commands`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

func runAdd(_ *cobra.Command, args []string) error {
	if err := auth.PromptAndVerifyPassword(); err != nil {
		return err
	}

	toolName := args[0]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("run 'zuko setup' first: %w", err)
	}

	if _, exists := cfg.Tools[toolName]; exists {
		return fmt.Errorf("%s is already configured — edit with 'zuko config'", toolName)
	}

	// Discover the real binary
	binaryPath, err := shim.DiscoverBinary(toolName, cfg.ShimDir)
	if err != nil {
		return err
	}

	// Build allow rules
	var allow [][]string
	switch {
	case addPassthrough:
		// Empty prefix matches everything
		allow = [][]string{{}}
	case len(addAllow) > 0:
		for _, entry := range addAllow {
			tokens := strings.Fields(entry)
			allow = append(allow, tokens)
		}
	default:
		return fmt.Errorf("specify --passthrough or --allow <subcommands>")
	}

	cfg.Tools[toolName] = config.Tool{
		RealBinary: binaryPath,
		AllowBare:  true,
		Allow:      allow,
		DenyFlags:  map[string][]string{},
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Create shim
	zukoPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine zuko binary path: %w", err)
	}
	if err := shim.Install(cfg.ShimDir, zukoPath, toolName); err != nil {
		return fmt.Errorf("failed to create shim: %w", err)
	}

	fmt.Printf("added %s (%s)\n", toolName, binaryPath)
	if addPassthrough {
		fmt.Println("  mode: passthrough (all commands allowed)")
	} else {
		fmt.Printf("  allowed: %s\n", strings.Join(addAllow, ", "))
	}
	fmt.Printf("  shim: %s/%s\n", cfg.ShimDir, toolName)
	return nil
}
