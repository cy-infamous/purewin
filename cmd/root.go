package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/cy-infamous/purewin/internal/core"
	"github.com/cy-infamous/purewin/internal/shell"
	"github.com/cy-infamous/purewin/internal/ui"
	"github.com/cy-infamous/purewin/internal/update"
)

var (
	// Global flags
	debug    bool
	dryRun   bool
	runAdmin bool
	noColor  bool
	autoYes  bool

	// Version info populated from main
	appVersion = "dev"
	appCommit  = "none"
	appDate    = "unknown"
)

// SetVersionInfo sets build-time version information.
func SetVersionInfo(version, commit, date string) {
	appVersion = version
	appCommit = commit
	appDate = date
}

var rootCmd = &cobra.Command{
	Use:   "pw",
	Short: "Deep clean and optimize your system",
	Long: `PureWin - Deep clean and optimize your system.

All-in-one toolkit for system cleanup, app uninstallation,
disk analysis, system optimization, and live monitoring.`,
}

// Execute runs the root command.
func Execute() error {
	// Clean up leftover .old binary from a previous self-update.
	update.CleanupOldBinary()

	// Enable Windows Virtual Terminal Processing globally so ANSI escape
	// codes render as colours in cmd.exe / PowerShell for ALL code paths
	// (inline spinners, confirm dialogs, styled fmt.Print output, etc.).
	vtSuccess := ui.EnableVTProcessing()

	// Warn if VT processing failed and we're in an interactive terminal
	if !vtSuccess && ui.IsTerminal() {
		fmt.Fprintln(os.Stderr, "WARNING: Your terminal does not support ANSI colors.")
		fmt.Fprintln(os.Stderr, "For the best experience, use Windows Terminal or PowerShell 7+.")
		fmt.Fprintln(os.Stderr, "PureWin will continue with limited styling.")
		fmt.Fprintln(os.Stderr, "")

		// Set NO_COLOR so lipgloss/termenv don't output garbled escape codes
		os.Setenv("NO_COLOR", "1")
	}

	return rootCmd.Execute()
}

func init() {
	// Assign Run in init() to break the initialization cycle between
	// rootCmd and runInteractiveMenu (which references rootCmd).
	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		runInteractiveMenu()
	}

	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Show detailed operation logs")
	rootCmd.PersistentFlags().BoolVar(&runAdmin, "admin", false, "Re-launch PureWin with elevated privileges")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().BoolVarP(&autoYes, "yes", "y", false, "Skip confirmation prompts (non-interactive mode)")

	// PersistentPreRun: if --admin is set, re-launch elevated and exit.
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if noColor {
			os.Setenv("NO_COLOR", "1")
		}

		if autoYes {
			ui.SetAutoYes(true)
		}

		if !runAdmin {
			return
		}
		// Already elevated — nothing to do.
		if core.IsElevated() {
			return
		}
		// Build args without --admin to avoid infinite loop.
		var elevatedArgs []string
		for _, a := range os.Args[1:] {
			if a != "--admin" {
				elevatedArgs = append(elevatedArgs, a)
			}
		}
		if err := core.RunElevated(elevatedArgs); err != nil {
			fmt.Fprintf(os.Stderr, "%s %v\n", ui.IconError, err)
			// Use panic recovery instead of os.Exit to allow defers to run.
			// Cobra will catch the panic from PersistentPreRun.
			rootCmd.SilenceErrors = true
			return
		}
		// sudo succeeded — parent process is done, child already ran.
		os.Exit(0)
	}

	// Register all subcommands
	rootCmd.AddCommand(cleanCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(optimizeCmd)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(purgeCmd)
	rootCmd.AddCommand(installerCmd)
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(whitelistCmd)
}

// runInteractiveShell launches the persistent interactive shell with
// slash-command autocomplete. The shell runs in a loop: each iteration
// runs a bubbletea program; when the user invokes a command, the shell
// exits, the command runs with full terminal control, then the shell
// relaunches with preserved state (output history, command history).
func runInteractiveShell() {
	if !ui.IsVTEnabled() {
		runSimpleShell()
		return
	}

	m := shell.NewShellModel(appVersion)

	// Add welcome output on first launch.
	m.AppendOutput("")

	for {
		p := tea.NewProgram(m)
		finalModel, err := p.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s Shell error: %v\n", ui.IconError, err)
			os.Exit(1)
		}

		result, ok := finalModel.(shell.ShellModel)
		if !ok {
			return
		}

		// User quit the shell entirely.
		if result.Quitting {
			return
		}

		// Command dispatch: run the cobra subcommand with full terminal control.
		if result.ExecCmd != "" {
			cmdArgs := append([]string{result.ExecCmd}, result.ExecArgs...)
			result.AppendOutput("")

			// Check if this command needs admin and we're not elevated.
			needsAdmin := false
			for _, cmd := range shell.AllCommands() {
				if cmd.Name == result.ExecCmd && cmd.AdminHint {
					needsAdmin = true
					break
				}
			}

			if needsAdmin && !core.IsElevated() && !dryRun {
				// Re-run via sudo so the user gets a password prompt
				// and sees the command output, then the TUI relaunches.
				exe, _ := os.Executable()
				sudoArgs := append([]string{exe}, cmdArgs...)
				sudoCmd := exec.Command("sudo", sudoArgs...)
				sudoCmd.Stdin = os.Stdin
				sudoCmd.Stdout = os.Stdout
				sudoCmd.Stderr = os.Stderr
				if err := sudoCmd.Run(); err != nil {
					result.AppendOutput(fmt.Sprintf("  %s Command failed: %v", ui.IconError, err))
				}
			} else {
				// Normal execution via cobra.
				rootCmd.SetArgs(cmdArgs)
				if err := rootCmd.Execute(); err != nil {
					result.AppendOutput("  Command failed: " + err.Error())
				}
			}

			result.AppendOutput("")

			// Clear the exec signal and relaunch shell.
			result.ExecCmd = ""
			result.ExecArgs = nil
		}

		// Preserve state for next iteration.
		m = result
	}
}

// runInteractiveMenu is kept for backward compatibility but now
// launches the interactive shell instead of the old menu.
func runInteractiveMenu() {
	runInteractiveShell()
}

// runSimpleShell provides a simple text-based REPL fallback when VT
// processing is unavailable (e.g., old Windows 10 cmd.exe without ANSI support).
func runSimpleShell() {
	fmt.Println()
	fmt.Printf("PureWin %s\n", appVersion)
	fmt.Println("Deep clean and optimize your system.")
	fmt.Println("Type /help for commands, /quit to exit.")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("pw> ")
		if !scanner.Scan() {
			break
		}
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" {
			continue
		}

		if !strings.HasPrefix(raw, "/") {
			fmt.Println("  Unknown input. Type /help for commands.")
			continue
		}

		parts := strings.Fields(raw[1:])
		if len(parts) == 0 {
			continue
		}

		cmdName := strings.ToLower(parts[0])
		args := parts[1:]

		switch cmdName {
		case "quit", "exit":
			fmt.Println("Goodbye!")
			return
		case "help":
			printSimpleHelp()
		case "version":
			fmt.Printf("  PureWin %s\n", appVersion)
		default:
			// Dispatch to cobra. Silence cobra's own error/usage output
			// to avoid double-printing in the REPL.
			rootCmd.SilenceErrors = true
			rootCmd.SilenceUsage = true
			cmdArgs := append([]string{cmdName}, args...)
			rootCmd.SetArgs(cmdArgs)
			if err := rootCmd.Execute(); err != nil {
				fmt.Printf("  Error: %s\n", err)
				fmt.Println("  Type /help for available commands.")
			}
			fmt.Println()
		}
	}
}

// printSimpleHelp displays command help for the simple shell fallback.
func printSimpleHelp() {
	fmt.Println()
	fmt.Println("  Available commands:")
	fmt.Println()
	fmt.Println("    /clean        Deep clean system caches and temp files")
	fmt.Println("    /uninstall    Remove installed applications")
	fmt.Println("    /optimize     Optimize system performance")
	fmt.Println("    /analyze      Explore disk space usage")
	fmt.Println("    /status       Live system health monitor")
	fmt.Println("    /purge        Clean project build artifacts")
	fmt.Println("    /installer    Find and remove old installer files")
	fmt.Println("    /update       Check for PureWin updates")
	fmt.Println("    /version      Show version info")
	fmt.Println("    /help         Show this help")
	fmt.Println("    /quit         Exit PureWin")
	fmt.Println()
}
