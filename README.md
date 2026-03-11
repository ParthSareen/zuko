# zuko

Read-only CLI sandbox for AI agents. Wraps tools like `gh` and `himalaya` behind an allowlist so agents can only run non-destructive commands.

## How it works

Zuko is a single Go binary that acts as a [multicall binary](https://en.wikipedia.org/wiki/Multicall_binary). When symlinked as `gh`, it intercepts the call, checks the command against an allowlist, and either proxies to the real binary or blocks it.

```
agent runs "gh issue list --state open"
        │
        ▼
~/.config/zuko/shims/gh  (symlink → zuko binary)
        │
        ▼
zuko loads ~/.config/zuko/config.yaml
        │
        ▼
["issue", "list"] matches allowlist → exec /opt/homebrew/bin/gh issue list --state open
```

```
agent runs "gh issue create --title oops"
        │
        ▼
zuko: blocked "gh issue create" — not in allowlist
```

## Install

```bash
go install github.com/ParthSareen/zuko@latest
```

Or build from source:

```bash
git clone https://github.com/ParthSareen/zuko.git
cd zuko
go build -o zuko .
cp zuko /usr/local/bin/zuko
```

## Setup

```bash
# Discover binaries on PATH, create shim symlinks, write default config
zuko setup

# Merge zuko settings into your existing openclaw.json
zuko init

# Or with dual enforcement (both zuko + openclaw allowlists)
zuko init --defense-in-depth
```

`zuko setup` creates symlinks in `~/.config/zuko/shims/` and writes a default config to `~/.config/zuko/config.yaml`.

`zuko init` prepends the shim directory to `env.PATH` in `~/.openclaw/openclaw.json` so the agent's `gh`, `himalaya`, etc. resolve to zuko's shims while all other tools (`git`, `node`, `python`, etc.) remain accessible.

## Config

The allowlist lives at `~/.config/zuko/config.yaml`:

```yaml
shim_dir: ~/.config/zuko/shims

tools:
  gh:
    real_binary: /opt/homebrew/bin/gh
    allow_bare: true
    allow:
      - ["issue", "list"]
      - ["issue", "view"]
      - ["pr", "list"]
      - ["pr", "view"]
      - ["pr", "diff"]
      - ["search", "issues"]
      - ["search", "code"]
      - ["api"]
      # ... see config.yaml for full list
    deny_flags:
      api: ["-X", "--method", "-f", "--raw-field", "-F", "--field", "--input"]

  himalaya:
    real_binary: /usr/local/bin/himalaya
    allow_bare: true
    allow:
      - ["envelope", "list"]
      - ["message", "read"]
      # ...
    deny_flags: {}
```

- **allow** — prefix match. `["issue", "list"]` permits `gh issue list --state open -R foo/bar`.
- **deny_flags** — per-subcommand flag blocklist. Blocks `gh api -X POST` while allowing `gh api /repos/...`.
- **allow_bare** — whether bare invocation (e.g. `gh` with no args) is permitted.

Edit the config to add new tools or commands. Requires authentication:

```bash
zuko config
```

## Authentication

Zuko uses your system password (macOS auth dialog / sudo on Linux) to gate privileged operations.

### Unlock (run unrestricted commands)

```bash
# Unlock for 5 minutes (default)
zuko unlock

# Unlock for 30 minutes
zuko unlock -d 30m

# Re-lock immediately
zuko lock
```

While unlocked, all shims pass commands through without filtering. The agent can't run `zuko unlock` because `zuko` itself isn't on the shim PATH.

### Protected operations

These commands require authentication:

| Command | What it does |
|---------|-------------|
| `zuko unlock` | Temporarily allow all commands through shims |
| `zuko config` | Open allowlist config in `$EDITOR` |
| `zuko init` | Merge settings into openclaw.json |
| `zuko add` | Add a new tool |
| `zuko remove` | Remove a tool |

## Adding and removing tools

```bash
# Passthrough (no subcommand filtering) — good for tools like jq, cat, rg
zuko add jq --passthrough

# Only allow specific subcommands
zuko add kubectl --allow get,describe,logs
zuko add docker --allow ps,images,logs

# Multi-word subcommand prefixes
zuko add docker --allow "container ls","image ls"

# Remove a tool
zuko remove jq
```

All `add`/`remove` operations require authentication. You can also fine-tune the config directly with `zuko config`.

## Commands

| Command | Description |
|---------|-------------|
| `zuko setup` | Discover binaries, create shim symlinks, write config |
| `zuko teardown` | Remove shim symlinks |
| `zuko init` | Merge zuko settings into openclaw.json |
| `zuko add` | Add a new CLI tool to the sandbox (requires auth) |
| `zuko remove` | Remove a CLI tool from the sandbox (requires auth) |
| `zuko config` | Edit allowlist config (requires auth) |
| `zuko unlock` | Temporarily allow all commands (requires auth) |
| `zuko lock` | Revoke unlock session |

## Platforms

macOS and Linux. Authentication uses the native macOS dialog on Darwin and `sudo -v` on Linux.
