package auth

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const touchIDSwift = `
import LocalAuthentication
import Foundation

let context = LAContext()
var error: NSError?

if context.canEvaluatePolicy(.deviceOwnerAuthenticationWithBiometrics, error: &error) {
    let semaphore = DispatchSemaphore(value: 0)
    var success = false
    context.evaluatePolicy(.deviceOwnerAuthenticationWithBiometrics, localizedReason: "Zuko: authenticate to proceed") { result, _ in
        success = result
        semaphore.signal()
    }
    semaphore.wait()
    if success {
        exit(0)
    }
}
exit(1)
`

const entitlementsPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>com.apple.security.personal-information.biometry</key>
    <true/>
</dict>
</plist>
`

func authHelperPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "zuko", "zuko-auth")
}

// buildAuthHelper compiles and codesigns the Touch ID helper binary.
// It caches the result and only rebuilds if the source changes.
func buildAuthHelper() (string, error) {
	helperPath := authHelperPath()
	srcHash := fmt.Sprintf("%x", sha256.Sum256([]byte(touchIDSwift)))

	// Check if cached binary is current
	hashPath := helperPath + ".sha256"
	if existing, err := os.ReadFile(hashPath); err == nil && string(existing) == srcHash {
		if _, err := os.Stat(helperPath); err == nil {
			return helperPath, nil
		}
	}

	dir := filepath.Dir(helperPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	// Write Swift source to temp file
	srcPath := helperPath + ".swift"
	if err := os.WriteFile(srcPath, []byte(touchIDSwift), 0600); err != nil {
		return "", err
	}
	defer os.Remove(srcPath)

	// Compile
	compile := exec.Command("swiftc", "-o", helperPath, "-framework", "LocalAuthentication", srcPath)
	compile.Stderr = os.Stderr
	if err := compile.Run(); err != nil {
		return "", fmt.Errorf("failed to compile auth helper: %w", err)
	}

	// Write entitlements
	entPath := helperPath + ".entitlements"
	if err := os.WriteFile(entPath, []byte(entitlementsPlist), 0600); err != nil {
		return "", err
	}
	defer os.Remove(entPath)

	// Codesign with entitlements
	sign := exec.Command("codesign", "--force", "--sign", "-", "--entitlements", entPath, helperPath)
	sign.Stderr = os.Stderr
	if err := sign.Run(); err != nil {
		return "", fmt.Errorf("failed to codesign auth helper: %w", err)
	}

	// Cache hash
	os.WriteFile(hashPath, []byte(srcHash), 0600)

	return helperPath, nil
}

// PromptAndVerifyPassword attempts Touch ID first (via compiled helper with
// biometry entitlement), then falls back to the macOS admin password dialog.
func PromptAndVerifyPassword() error {
	// Try Touch ID via compiled, signed helper
	if helperPath, err := buildAuthHelper(); err == nil {
		cmd := exec.Command(helperPath)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	// Fallback: osascript admin dialog (works on all Macs without swift/Xcode)
	cmd := exec.Command("osascript", "-e",
		`do shell script "echo ok" with administrator privileges with prompt "Zuko: authenticate to proceed"`)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("authentication failed")
	}
	return nil
}
