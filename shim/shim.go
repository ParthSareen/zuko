package shim

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Install creates a symlink in shimDir pointing toolName → zukoPath.
func Install(shimDir, zukoPath, toolName string) error {
	if err := os.MkdirAll(shimDir, 0755); err != nil {
		return err
	}
	link := filepath.Join(shimDir, toolName)
	// Remove existing symlink if present
	if _, err := os.Lstat(link); err == nil {
		os.Remove(link)
	}
	return os.Symlink(zukoPath, link)
}

// Remove deletes a shim symlink.
func Remove(shimDir, toolName string) error {
	return os.Remove(filepath.Join(shimDir, toolName))
}

// IsZukoShim checks if the path is a symlink pointing to zukoPath.
func IsZukoShim(path, zukoPath string) bool {
	target, err := os.Readlink(path)
	if err != nil {
		return false
	}
	absTarget, _ := filepath.Abs(target)
	absZuko, _ := filepath.Abs(zukoPath)
	return absTarget == absZuko
}

// DiscoverBinary finds the real binary for toolName on PATH, skipping shimDir.
func DiscoverBinary(toolName, shimDir string) (string, error) {
	pathEnv := os.Getenv("PATH")
	absShimDir, _ := filepath.Abs(shimDir)

	for _, dir := range strings.Split(pathEnv, string(os.PathListSeparator)) {
		absDir, _ := filepath.Abs(dir)
		if absDir == absShimDir {
			continue
		}
		candidate := filepath.Join(dir, toolName)
		info, err := os.Stat(candidate)
		if err != nil {
			continue
		}
		if info.Mode().IsRegular() && info.Mode()&0111 != 0 {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("%s not found on PATH", toolName)
}
