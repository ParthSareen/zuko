package cmd

import (
	"fmt"
	"strings"

	"github.com/ParthSareen/zuko/auth"
	"github.com/ParthSareen/zuko/remote"
	"github.com/spf13/cobra"
)

var pairName string

func init() {
	pairCmd.Flags().StringVar(&pairName, "name", "", "name for the paired client")
	rootCmd.AddCommand(pairCmd)
}

var pairCmd = &cobra.Command{
	Use:   "pair [name]",
	Short: "Pair a remote approval client",
	Long:  "Creates a bearer token for a remote client that can approve one-shot locked command requests from zuko serve.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runPair,
}

func runPair(_ *cobra.Command, args []string) error {
	name := strings.TrimSpace(pairName)
	if name == "" && len(args) > 0 {
		name = strings.TrimSpace(args[0])
	}
	if name == "" {
		name = "iPhone"
	}

	if err := auth.PromptAndVerifyPassword("pair remote client " + name); err != nil {
		return err
	}

	client, token, err := remote.PairClient(name)
	if err != nil {
		return fmt.Errorf("failed to pair client: %w", err)
	}

	fmt.Printf("Paired %s (%s).\n", client.Name, client.ID)
	fmt.Println()
	fmt.Println("Save this token in the remote client. It is shown once:")
	fmt.Printf("  %s\n", token)
	fmt.Println()

	if state, err := remote.LoadServeState(); err == nil {
		fmt.Printf("Server: %s\n", state.URL)
		fmt.Println()
		fmt.Println("Test with:")
		fmt.Printf("  curl -H 'Authorization: Bearer %s' %s/v1/approvals\n", token, strings.TrimRight(state.URL, "/"))
	} else {
		fmt.Println("Start the approval server with:")
		fmt.Println("  zuko serve --tailscale")
	}

	return nil
}
