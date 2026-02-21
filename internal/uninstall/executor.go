package uninstall

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"
)

const (
	// uninstallTimeout is the maximum time to wait for an uninstall process.
	uninstallTimeout = 120 * time.Second
)

// msiGUIDPattern matches MSI product GUIDs like {XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX}.
var msiGUIDPattern = regexp.MustCompile(`\{[0-9A-Fa-f-]+\}`)

// ─── Public API ──────────────────────────────────────────────────────────────

// UninstallApp executes the uninstall command for the given application.
// If quiet is true and a QuietUninstallString is available, it is preferred.
// The process is given a 120-second timeout.
func UninstallApp(app InstalledApp, quiet bool) error {
	cmdStr := chooseUninstallCommand(app, quiet)
	if cmdStr == "" {
		return fmt.Errorf("no uninstall command found for %q", app.Name)
	}

	// Detect MSI-based uninstalls and handle them specially.
	if isMSIUninstall(cmdStr) {
		return runMSIUninstall(cmdStr, quiet)
	}

	return runUninstallCommand(cmdStr)
}

// ─── Internal Helpers ────────────────────────────────────────────────────────

// chooseUninstallCommand selects the appropriate uninstall string.
func chooseUninstallCommand(app InstalledApp, quiet bool) string {
	if quiet && app.QuietUninstallString != "" {
		return app.QuietUninstallString
	}
	return app.UninstallString
}

// isMSIUninstall returns true if the command invokes msiexec.
func isMSIUninstall(cmd string) bool {
	return strings.Contains(strings.ToLower(cmd), "msiexec")
}

// runMSIUninstall extracts the GUID and runs msiexec with proper flags.
func runMSIUninstall(cmdStr string, quiet bool) error {
	guid := msiGUIDPattern.FindString(cmdStr)
	if guid == "" {
		// Fallback to running the raw command if we can't parse the GUID.
		return runUninstallCommand(cmdStr)
	}

	args := []string{"/x", guid}
	if quiet {
		args = append(args, "/qn", "/norestart")
	}

	ctx, cancel := context.WithTimeout(context.Background(), uninstallTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "msiexec.exe", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return handleExitError(err, output)
	}
	return nil
}

// parseExePath extracts the executable path from an uninstall command string.
// Handles both quoted paths ("C:\Program Files\app.exe" /S) and unquoted
// paths (C:\app\uninstall.exe /silent).
func parseExePath(cmdStr string) string {
	cmdStr = strings.TrimSpace(cmdStr)
	if cmdStr == "" {
		return ""
	}

	// Quoted executable: "C:\Program Files\App\uninstall.exe" /args
	if cmdStr[0] == '"' {
		end := strings.Index(cmdStr[1:], `"`)
		if end >= 0 {
			return cmdStr[1 : end+1]
		}
		return ""
	}

	// Unquoted: find the .exe boundary (case-insensitive).
	lower := strings.ToLower(cmdStr)
	if idx := strings.Index(lower, ".exe"); idx >= 0 {
		return cmdStr[:idx+4]
	}

	// Fallback: first space-delimited token.
	if i := strings.IndexByte(cmdStr, ' '); i >= 0 {
		return cmdStr[:i]
	}
	return cmdStr
}

// runUninstallCommand executes an uninstall command string.
// It first attempts direct execution (without cmd.exe) to prevent shell
// metacharacter injection (e.g., & | > < chaining). Only falls back to
// cmd /C when the executable can't be resolved on disk.
func runUninstallCommand(cmdStr string) error {
	ctx, cancel := context.WithTimeout(context.Background(), uninstallTimeout)
	defer cancel()

	// Attempt direct execution: parse the exe path and verify it exists.
	// This prevents command injection because CreateProcess does not
	// interpret shell metacharacters like & | > <.
	exe := parseExePath(cmdStr)
	if exe != "" {
		if _, statErr := os.Stat(exe); statErr == nil {
			cmd := exec.CommandContext(ctx, exe)
			cmd.SysProcAttr = &syscall.SysProcAttr{
				CmdLine: cmdStr, // Pass the full command line verbatim.
			}
			output, err := cmd.CombinedOutput()
			if err != nil {
				return handleExitError(err, output)
			}
			return nil
		}
	}

	// Fallback: use cmd /C for commands where the executable can't be
	// resolved (e.g., PATH-relative executables). Most legitimate uninstall
	// strings use absolute paths, so this path should be rare.
	cmd := exec.CommandContext(ctx, "cmd.exe", "/C", cmdStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return handleExitError(err, output)
	}
	return nil
}

// handleExitError wraps an exec error with contextual information.
// Common MSI exit codes are translated to human-readable messages.
func handleExitError(err error, output []byte) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("uninstall timed out after %s", uninstallTimeout)
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		code := exitErr.ExitCode()
		switch code {
		case 1605:
			return fmt.Errorf("product is not currently installed (exit code 1605)")
		case 1641:
			// Restart required but uninstall itself succeeded.
			return fmt.Errorf("uninstall succeeded — restart required (exit code 1641)")
		case 3010:
			// Restart required but uninstall itself succeeded.
			return fmt.Errorf("uninstall succeeded — restart required (exit code 3010)")
		default:
			outputStr := strings.TrimSpace(string(output))
			if len(outputStr) > 200 {
				outputStr = outputStr[:200] + "..."
			}
			if outputStr != "" {
				return fmt.Errorf("uninstall failed (exit code %d): %s", code, outputStr)
			}
			return fmt.Errorf("uninstall failed (exit code %d)", code)
		}
	}

	return fmt.Errorf("uninstall command error: %w", err)
}
