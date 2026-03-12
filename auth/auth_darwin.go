package auth

import (
	"fmt"
	"os"
	"os/exec"
)

const touchIDSwift = `
import LocalAuthentication
import Foundation

let context = LAContext()
var error: NSError?

if context.canEvaluatePolicy(.deviceOwnerAuthentication, error: &error) {
    let semaphore = DispatchSemaphore(value: 0)
    var success = false
    context.evaluatePolicy(.deviceOwnerAuthentication, localizedReason: "Zuko: authenticate to proceed") { result, _ in
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

// PromptAndVerifyPassword attempts Touch ID first (via LocalAuthentication),
// then falls back to the macOS admin password dialog.
func PromptAndVerifyPassword() error {
	// Try LocalAuthentication (Touch ID → Apple Watch → password)
	cmd := exec.Command("swift", "-e", touchIDSwift)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err == nil {
		return nil
	}

	// Fallback: osascript admin dialog (works on all Macs without swift/Xcode)
	cmd = exec.Command("osascript", "-e",
		`do shell script "echo ok" with administrator privileges with prompt "Zuko: authenticate to proceed"`)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("authentication failed")
	}
	return nil
}
