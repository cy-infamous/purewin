package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/lakshaymaurya-felt/winmole/internal/ui"
)

var (
	// Global flags
	debug  bool
	dryRun bool

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
	Use:   "wm",
	Short: "Deep clean and optimize your Windows",
	Long: `WinMole - Deep clean and optimize your Windows.

A Windows port of Mole (https://github.com/tw93/Mole).
All-in-one toolkit for system cleanup, app uninstallation,
disk analysis, system optimization, and live monitoring.`,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Assign Run in init() to break the initialization cycle between
	// rootCmd and runInteractiveMenu (which references rootCmd).
	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		runInteractiveMenu()
	}

	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Show detailed operation logs")

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
}

// runInteractiveMenu launches the full-screen interactive main menu.
// It shows the mole intro animation, then enters a bubbletea alt-screen
// menu. When the user selects a command, it exits bubbletea and executes
// the corresponding cobra subcommand.
func runInteractiveMenu() {
	// Show the mole intro animation on launch.
	ui.ShowMoleIntro()

	// Run the interactive menu.
	selected, err := runMainMenu()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s Menu error: %v\n", ui.IconError, err)
		os.Exit(1)
	}

	// User quit without selecting — clean exit.
	if selected == "" {
		return
	}

	// Execute the selected subcommand via cobra.
	rootCmd.SetArgs([]string{selected})
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
