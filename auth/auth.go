package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const DefaultUnlockDuration = 5 * time.Minute

type Grant struct {
	Expires time.Time `json:"expires"`
	Token   string    `json:"token"`
}

type UnlockState struct {
	Global *Grant            `json:"global,omitempty"`
	Grants map[string]*Grant `json:"grants,omitempty"`
}

func unlockPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "zuko", "unlock.json")
}

func loadState() *UnlockState {
	data, err := os.ReadFile(unlockPath())
	if err != nil {
		return &UnlockState{}
	}

	// Try new format first
	var state UnlockState
	if err := json.Unmarshal(data, &state); err != nil {
		return &UnlockState{}
	}

	// Backward compat: old flat format had top-level "expires" and "token"
	if state.Global == nil && state.Grants == nil {
		var legacy struct {
			Expires time.Time `json:"expires"`
			Token   string    `json:"token"`
		}
		if err := json.Unmarshal(data, &legacy); err == nil && legacy.Token != "" {
			state.Global = &Grant{Expires: legacy.Expires, Token: legacy.Token}
		}
	}

	return &state
}

func saveState(state *UnlockState) error {
	dir := filepath.Dir(unlockPath())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(unlockPath(), data, 0600)
}

func newGrant(duration time.Duration) (*Grant, error) {
	token := make([]byte, 16)
	if _, err := rand.Read(token); err != nil {
		return nil, err
	}
	return &Grant{
		Expires: time.Now().Add(duration),
		Token:   hex.EncodeToString(token),
	}, nil
}

func grantValid(g *Grant) bool {
	return g != nil && time.Now().Before(g.Expires)
}

// IsUnlocked checks if there is a valid global unlock session.
func IsUnlocked() bool {
	state := loadState()
	return grantValid(state.Global)
}

// IsGranted checks if the given scope is unlocked.
// It checks global first, then exact (e.g. "git:commit amend"),
// then any shorter subcommand prefix (e.g. "git:commit"), then tool-level ("git").
func IsGranted(scope string) bool {
	state := loadState()

	// Global unlock covers everything
	if grantValid(state.Global) {
		return true
	}

	if state.Grants == nil {
		return false
	}

	// Exact scope match (e.g. "git:commit amend")
	if g, ok := state.Grants[scope]; ok && grantValid(g) {
		return true
	}

	parts := strings.SplitN(scope, ":", 2)
	if len(parts) != 2 {
		return false
	}
	tool := parts[0]

	// Walk subcommand prefixes from most to least specific:
	// "gh:issue edit" -> check "gh:issue"
	subcmds := strings.Fields(parts[1])
	for i := len(subcmds) - 1; i >= 1; i-- {
		prefix := tool + ":" + strings.Join(subcmds[:i], " ")
		if g, ok := state.Grants[prefix]; ok && grantValid(g) {
			return true
		}
	}

	// Tool-level match: "git" covers "git:commit"
	if g, ok := state.Grants[tool]; ok && grantValid(g) {
		return true
	}

	return false
}

// Unlock creates a global unlock session (existing behavior).
func Unlock(duration time.Duration) error {
	state := loadState()
	g, err := newGrant(duration)
	if err != nil {
		return err
	}
	state.Global = g
	return saveState(state)
}

// UnlockScope adds a scoped grant.
func UnlockScope(scope string, duration time.Duration) error {
	state := loadState()
	g, err := newGrant(duration)
	if err != nil {
		return err
	}
	if state.Grants == nil {
		state.Grants = make(map[string]*Grant)
	}
	state.Grants[scope] = g
	return saveState(state)
}

// Lock removes the global unlock session and all grants.
func Lock() error {
	err := os.Remove(unlockPath())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// LockScope removes a specific scoped grant. If scope matches a tool name,
// all grants prefixed with that tool are removed.
func LockScope(scope string) error {
	state := loadState()

	if state.Grants == nil {
		return nil
	}

	// If scope has no colon, it's a tool-level lock — remove all grants for that tool
	if !strings.Contains(scope, ":") {
		for key := range state.Grants {
			if key == scope || strings.HasPrefix(key, scope+":") {
				delete(state.Grants, key)
			}
		}
	} else {
		delete(state.Grants, scope)
	}

	return saveState(state)
}
