package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cy-infamous/purewin/internal/ui"
	"github.com/cy-infamous/purewin/internal/uninstall"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall [path]",
	Short: "Remove apps completely",
	Long: `Thoroughly remove applications along with their registry entries, data, and hidden remnants.

Defaults to showing only apps installed under the current drive/directory.
Use --all to show all installed applications regardless of location.

Examples:
  pw uninstall              Show apps installed on the current drive
  pw uninstall D:\Programs  Show apps installed under a specific path
  pw uninstall --all        Show all installed applications`,
	Args: cobra.MaximumNArgs(1),
	Run:  runUninstall,
}

func init() {
	uninstallCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview without uninstalling")
	uninstallCmd.Flags().Bool("all", false, "Show all installed apps regardless of location")
	uninstallCmd.Flags().Bool("quiet", false, "Prefer silent uninstall commands")
	uninstallCmd.Flags().Bool("show-all", false, "Show system components too")
	uninstallCmd.Flags().String("search", "", "Search for apps by name")
}

func runUninstall(cmd *cobra.Command, args []string) {
	quiet, _ := cmd.Flags().GetBool("quiet")
	allFlag, _ := cmd.Flags().GetBool("all")
	showAll, _ := cmd.Flags().GetBool("show-all")
	search, _ := cmd.Flags().GetString("search")

	// Determine filter path.
	var filterPath string
	if len(args) > 0 {
		filterPath = args[0]
	} else if !allFlag {
		cwd, cwdErr := os.Getwd()
		if cwdErr != nil {
			fmt.Println(ui.ErrorStyle().Render(
				fmt.Sprintf("  %s Cannot determine current directory: %v", ui.IconError, cwdErr)))
			os.Exit(1)
		}
		filterPath = cwd
	}

	// Scan installed apps from the registry.
	fmt.Println()
	spin := ui.NewInlineSpinner()
	spin.Start("Scanning installed applications...")

	apps, err := uninstall.GetInstalledApps(showAll)
	if err != nil {
		spin.StopWithError(fmt.Sprintf("Failed to read registry: %s", err))
		os.Exit(1)
	}

	// Filter to apps under the target path (unless --all).
	if filterPath != "" {
		apps = uninstall.FilterByPath(apps, filterPath)
		spin.Stop(fmt.Sprintf("Found %d applications under %s", len(apps), filterPath))
	} else {
		spin.Stop(fmt.Sprintf("Found %d installed applications", len(apps)))
	}

	// Apply search filter if specified.
	if search != "" {
		apps = filterAppsByName(apps, search)
		if len(apps) == 0 {
			fmt.Println(ui.WarningStyle().Render(
				fmt.Sprintf("  No applications matching %q found.", search)))
			return
		}
		fmt.Println(ui.InfoStyle().Render(
			fmt.Sprintf("  %d application(s) matching %q", len(apps), search)))
	}

	// Quick single-app uninstall if --quiet + --search yields exactly one result.
	if quiet && search != "" && len(apps) == 1 {
		runSingleUninstall(apps[0], dryRun, quiet)
		return
	}

	// Batch uninstall flow with selector.
	if err := uninstall.RunBatchUninstall(apps, dryRun); err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %s\n",
			ui.ErrorStyle().Render(ui.IconError),
			ui.ErrorStyle().Render(err.Error()))
		os.Exit(1)
	}
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
func runSingleUninstall(app uninstall.InstalledApp, dryRun bool, quiet bool) {
	if dryRun {
		fmt.Printf("\n  DRY RUN: Would uninstall %s\n", app.Name)
		return
	}

	confirmed, err := ui.Confirm(fmt.Sprintf("Uninstall %s?", app.Name))
	if err != nil || !confirmed {
		fmt.Println(ui.MutedStyle().Render("  Cancelled."))
		return
	}

	spin := ui.NewInlineSpinner()
	spin.Start(fmt.Sprintf("Uninstalling %s...", app.Name))

	if uninstErr := uninstall.UninstallApp(app, quiet); uninstErr != nil {
		spin.StopWithError(fmt.Sprintf("Failed: %s", uninstErr))
		os.Exit(1)
	}
	spin.Stop(fmt.Sprintf("Uninstalled %s", app.Name))
}
