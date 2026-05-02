//go:build linux

package core

import (
	"fmt"
	"os"
	"os/exec"
)

// IsElevated returns true if the current process is running as root.
func IsElevated() bool {
	return os.Getuid() == 0
}

// RequireAdmin returns an error if the current process is not root.
func RequireAdmin(operation string) error {
	if IsElevated() {
		return nil
	}
	return fmt.Errorf(
		"operation %q requires root privileges\n"+
			"  → Re-run with: sudo pw %s",
		operation, operation,
	)
}

// RunElevated re-launches the current process with sudo.
func RunElevated(args []string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine executable path: %w", err)
	}

	allArgs := append([]string{exe}, args...)
	cmd := exec.Command("sudo", allArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
