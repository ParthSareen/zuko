package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const DefaultUnlockDuration = 5 * time.Minute

type UnlockState struct {
	Expires time.Time `json:"expires"`
	Token   string    `json:"token"`
}

func unlockPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "zuko", "unlock.json")
}

// IsUnlocked checks if there is a valid, non-expired unlock session.
func IsUnlocked() bool {
	data, err := os.ReadFile(unlockPath())
	if err != nil {
		return false
	}
	var state UnlockState
	if err := json.Unmarshal(data, &state); err != nil {
		return false
	}
	if time.Now().After(state.Expires) {
		os.Remove(unlockPath())
		return false
	}
	return true
}

// Unlock creates an unlock session that expires after the given duration.
func Unlock(duration time.Duration) error {
	token := make([]byte, 16)
	if _, err := rand.Read(token); err != nil {
		return err
	}

	state := UnlockState{
		Expires: time.Now().Add(duration),
		Token:   hex.EncodeToString(token),
	}

	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	dir := filepath.Dir(unlockPath())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(unlockPath(), data, 0600)
}

// Lock removes the unlock session immediately.
func Lock() error {
	err := os.Remove(unlockPath())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
