package core

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/windows"
)

// IsElevated returns true if the current process is running with
// administrator privileges.
func IsElevated() bool {
	token := windows.GetCurrentProcessToken()
	return token.IsElevated()
}

// RequireAdmin returns an error if the current process is not elevated.
// The operation parameter is included in the error message for context.
func RequireAdmin(operation string) error {
	if IsElevated() {
		return nil
	}
	return fmt.Errorf(
		"operation %q requires administrator privileges\n"+
			"  → Re-run with: pw %s --admin\n"+
			"  → Or right-click Terminal → Run as Administrator",
		operation, operation,
	)
}

// escapeWindowsArg escapes a single command-line argument for Windows
// using the CommandLineToArgvW convention. Arguments containing spaces,
// tabs, or quotes are wrapped in double quotes with proper backslash
// escaping to prevent argument injection during elevation.
func escapeWindowsArg(arg string) string {
	if arg == "" {
		return `""`
	}
	// If no special characters, return as-is.
	if !strings.ContainsAny(arg, " \t\"\\") {
		return arg
	}

	var b strings.Builder
	b.WriteByte('"')
	nBackslash := 0
	for _, c := range arg {
		switch c {
		case '\\':
			nBackslash++
		case '"':
			// Double existing backslashes + escape the quote.
			for i := 0; i < nBackslash*2+1; i++ {
				b.WriteByte('\\')
			}
			b.WriteByte('"')
			nBackslash = 0
		default:
			// Flush accumulated backslashes.
			for i := 0; i < nBackslash; i++ {
				b.WriteByte('\\')
			}
			b.WriteRune(c)
			nBackslash = 0
		}
	}
	// Trailing backslashes must be doubled before the closing quote.
	for i := 0; i < nBackslash*2; i++ {
		b.WriteByte('\\')
	}
	b.WriteByte('"')
	return b.String()
}

// RunElevated re-launches the current process with administrator privileges
// via the Windows ShellExecuteW "runas" verb. This triggers a UAC prompt.
// The current process exits after launching the elevated one.
// The args parameter should contain the command-line arguments to pass
// (excluding the --admin flag itself to avoid an infinite re-launch loop).
func RunElevated(args []string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine executable path: %w", err)
	}

	// Convert exe path and args to UTF16 for ShellExecuteW.
	exeUTF16, err := windows.UTF16PtrFromString(exe)
	if err != nil {
		return fmt.Errorf("invalid executable path: %w", err)
	}

	// Properly escape each argument to prevent injection via spaces/quotes.
	escaped := make([]string, len(args))
	for i, arg := range args {
		escaped[i] = escapeWindowsArg(arg)
	}
	argStr := strings.Join(escaped, " ")
	argsUTF16, err := windows.UTF16PtrFromString(argStr)
	if err != nil {
		return fmt.Errorf("invalid arguments: %w", err)
	}

	verbUTF16, _ := windows.UTF16PtrFromString("runas")

	// ShellExecuteW with "runas" triggers UAC. Returns error if ret <= 32.
	err = windows.ShellExecute(0, verbUTF16, exeUTF16, argsUTF16, nil, windows.SW_SHOWNORMAL)
	if err != nil {
		return fmt.Errorf("UAC elevation failed: %w", err)
	}

	// Elevated process launched successfully — exit the current one.
	os.Exit(0)
	return nil // unreachable
}
