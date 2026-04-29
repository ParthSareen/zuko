package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ParthSareen/zuko/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(clankerCmd)
}

var clankerCmd = &cobra.Command{
	Use:     "clanker",
	Aliases: []string{"usage"},
	Short:   "Print LLM-targeted usage instructions for the current sandbox",
	Long:    "Prints a self-contained brief explaining how zuko gates CLI tools, plus the effective allow/locked lists from this machine's config. Pipe into an agent's context or system prompt.",
	Run:     runClanker,
}

func runClanker(_ *cobra.Command, _ []string) {
	fmt.Print(clankerPreamble)
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("\n(config not loaded: %v)\n", err)
		return
	}
	fmt.Print(renderConfigDigest(cfg))
}

const clankerPreamble = `# Zuko sandbox brief (for AI agents)

You are running inside a machine where some CLI tools are wrapped by zuko, a
read-only sandbox. The wrapped tools (see list below) resolve through symlinks
in ~/.config/zuko/shims/ and consult an allowlist before executing. Unwrapped
tools behave normally.

## What happens when you run a wrapped command

1. Allowed     -> command runs normally. No special handling.
2. Blocked     -> stderr says: "<tool> <args> is not in allowlist". Process
                  exits 1. STOP. Do not retry, do not try variants. Tell the
                  user the command is blocked and surface the 'zuko unlock ...'
                  hint zuko prints. Wait for them to unlock, then re-run.
3. Locked      -> Touch ID prompt fires. If no human is present the prompt
                  fails and the command exits 1. STOP and surface the
                  'zuko unlock <tool> <subcmd>' hint to the user.
4. Denied flag -> e.g. "gh api -X POST" blocked even though "gh api" is in
                  the allowlist. STOP. Do not retry with a different flag
                  shape to dodge the deny. Surface to the user.

After a successful 'zuko unlock <tool> <subcmd>' the lock is open for a
time-limited, scoped grant. Re-issue the original command verbatim.

## Hard rules

- DO NOT bypass zuko by invoking the real binary directly. Specifically: do
  not run /usr/bin/git, /opt/homebrew/bin/gh, /opt/homebrew/bin/<tool>, etc.
  Do not set PATH=/usr/bin to skip the shim. Do not 'cd /usr/bin && ./git'.
  Do not assemble the path through 'sh -c' or printf. Do not symlink the
  real binary elsewhere. Any of these is a sandbox-evasion attempt and is
  not authorized.
- DO NOT run 'zuko unlock', 'zuko lock', 'zuko config', 'zuko add',
  'zuko remove', 'zuko init', 'zuko setup', 'zuko teardown', or
  'zuko timeout'. These require Touch ID and are user-only.
- DO NOT edit ~/.config/zuko/config.yaml directly.
- DO use 'zuko clanker' (this command) and 'zuko version' freely - they are
  read-only.

## How to ask the user for an unlock

Whenever a wrapped command is blocked or locked, zuko prints the exact
unlock command to run and copies it to the clipboard. Pass that string to
the user and pause. Example:

    zuko: git commit requires unlock - run 'zuko unlock git commit'
    (copied to clipboard)

Tell the user: "git commit is locked. Run 'zuko unlock git commit' in your
terminal, then I'll retry." Do not invent unlock commands - use the one
zuko printed.

`

func renderConfigDigest(cfg *config.Config) string {
	var b strings.Builder
	b.WriteString("## Effective rules on this machine\n\n")

	names := make([]string, 0, len(cfg.Tools))
	for name := range cfg.Tools {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		tool := cfg.Tools[name]
		fmt.Fprintf(&b, "### %s\n", name)
		if tool.AllowAll {
			b.WriteString("- mode: allow_all (everything passes through except the locked list below)\n")
		} else {
			fmt.Fprintf(&b, "- mode: allowlist (allow_bare=%v)\n", tool.AllowBare)
		}

		if len(tool.Allow) > 0 {
			b.WriteString("- allow (passes through, no prompt):\n")
			for _, p := range tool.Allow {
				fmt.Fprintf(&b, "    - %s %s\n", name, strings.Join(p, " "))
			}
		}
		if len(tool.Locked) > 0 {
			b.WriteString("- locked (Touch ID required - ask user to 'zuko unlock'):\n")
			for _, p := range tool.Locked {
				fmt.Fprintf(&b, "    - %s %s\n", name, strings.Join(p, " "))
			}
		}
		if len(tool.DenyFlags) > 0 {
			b.WriteString("- deny_flags (blocked even on allowed subcommands):\n")
			subs := make([]string, 0, len(tool.DenyFlags))
			for s := range tool.DenyFlags {
				subs = append(subs, s)
			}
			sort.Strings(subs)
			for _, s := range subs {
				fmt.Fprintf(&b, "    - %s %s: %s\n", name, s, strings.Join(tool.DenyFlags[s], " "))
			}
		}
		b.WriteString("\n")
	}

	b.WriteString("Anything not listed above for a wrapped tool will be blocked. Anything not listed as a tool at all is unwrapped and runs normally.\n")
	return b.String()
}
