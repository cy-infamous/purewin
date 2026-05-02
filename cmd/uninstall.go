package cmd

import (
	"fmt"
	"runtime"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/lakshaymaurya-felt/purewin/internal/core"
	"github.com/lakshaymaurya-felt/purewin/internal/ui"
	"github.com/lakshaymaurya-felt/purewin/internal/uninstall"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove apps completely",
	Long:  "Remove installed applications and their associated data.",
	RunE:  runUninstall,
}

func init() {
	uninstallCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview without uninstalling")
	uninstallCmd.Flags().Bool("quiet", false, "Prefer silent uninstall commands")
	uninstallCmd.Flags().Bool("show-all", false, "Show system components too")
	uninstallCmd.Flags().String("search", "", "Search for apps by name")
}

func runUninstall(cmd *cobra.Command, args []string) error {
	// Check if running as administrator and warn if not.
	if !core.IsElevated() {
		elevateHint := "pw --admin uninstall"
		if runtime.GOOS == "linux" {
			elevateHint = "sudo pw uninstall"
		}
		fmt.Println(ui.WarningStyle().Render(
			"  ⚠ Not running as administrator\n" +
				"  Some apps may require elevated privileges to uninstall.\n" +
				fmt.Sprintf("  If uninstall fails, try: %s", elevateHint)))
		fmt.Println()
	}

	quiet, _ := cmd.Flags().GetBool("quiet")
	showAll, _ := cmd.Flags().GetBool("show-all")
	search, _ := cmd.Flags().GetString("search")

	// Scan installed apps from the registry.
	fmt.Println()
	spin := ui.NewInlineSpinner()
	spin.Start("Scanning installed applications...")

	apps, err := uninstall.GetInstalledApps(showAll)
	if err != nil {
		spin.StopWithError(fmt.Sprintf("Failed to read registry: %s", err))
		return fmt.Errorf("failed to read registry: %w", err)
	}
	spin.Stop(fmt.Sprintf("Found %d installed applications", len(apps)))

	// Apply search filter if specified.
	if search != "" {
		apps = filterAppsByName(apps, search)
		if len(apps) == 0 {
			fmt.Println(ui.WarningStyle().Render(
				fmt.Sprintf("  No applications matching %q found.", search)))
			return nil
		}
		fmt.Println(ui.InfoStyle().Render(
			fmt.Sprintf("  %d application(s) matching %q", len(apps), search)))
	}

	// Quick single-app uninstall if --quiet + --search yields exactly one result.
	if quiet && search != "" && len(apps) == 1 {
		if err := runSingleUninstall(apps[0], dryRun, quiet); err != nil {
			return err
		}
		return nil
	}

	// Batch uninstall flow with selector.
	if err := uninstall.RunBatchUninstall(apps, dryRun); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			ui.ErrorStyle().Render(ui.IconError),
			ui.ErrorStyle().Render(err.Error()))
		return fmt.Errorf("batch uninstall failed: %w", err)
	}
	return nil
}

// filterAppsByName returns apps whose Name contains the search term
// (case-insensitive).
func filterAppsByName(apps []uninstall.InstalledApp, search string) []uninstall.InstalledApp {
	lower := strings.ToLower(search)
	var filtered []uninstall.InstalledApp
	for _, app := range apps {
		if strings.Contains(strings.ToLower(app.Name), lower) {
			filtered = append(filtered, app)
		}
	}
	return filtered
}

// runSingleUninstall handles uninstalling a single app directly.
func runSingleUninstall(app uninstall.InstalledApp, dryRun bool, quiet bool) error {
	if dryRun {
		fmt.Printf("\n  DRY RUN: Would uninstall %s\n", app.Name)
		return nil
	}

	confirmed, err := ui.Confirm(fmt.Sprintf("Uninstall %s?", app.Name))
	if err != nil || !confirmed {
		fmt.Println(ui.MutedStyle().Render("  Cancelled."))
		return nil
	}

	spin := ui.NewInlineSpinner()
	spin.Start(fmt.Sprintf("Uninstalling %s...", app.Name))

	if uninstErr := uninstall.UninstallApp(app, quiet); uninstErr != nil {
		spin.StopWithError(fmt.Sprintf("Failed: %s", uninstErr))
		return fmt.Errorf("failed to uninstall %s: %w", app.Name, uninstErr)
	}
	spin.Stop(fmt.Sprintf("Uninstalled %s", app.Name))
	return nil
}
