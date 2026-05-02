package cmd

import (
	"fmt"
	"sort"
	"time"

	"github.com/lakshaymaurya-felt/purewin/internal/core"
	"github.com/lakshaymaurya-felt/purewin/internal/installer"
	"github.com/lakshaymaurya-felt/purewin/internal/ui"
	"github.com/lakshaymaurya-felt/purewin/internal/util"
	"github.com/spf13/cobra"
)

var installerCmd = &cobra.Command{
	Use:   "installer",
	Short: "Find and remove installer files",
	Long:  "Scan Downloads, Desktop, and package manager caches for installer files (.exe, .msi, .msix).",
	RunE:  runInstaller,
}

func init() {
	installerCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview without deleting")
	installerCmd.Flags().Int("min-age", 0, "Minimum file age in days")
	installerCmd.Flags().String("min-size", "", "Minimum file size (e.g., 10MB)")
}

func runInstaller(cmd *cobra.Command, args []string) error {
	// Parse flags
	minAge, _ := cmd.Flags().GetInt("min-age")
	minSizeStr, _ := cmd.Flags().GetString("min-size")

	var minSize int64
	if minSizeStr != "" {
		size, err := util.ParseSize(minSizeStr)
		if err != nil {
			return fmt.Errorf("invalid size format: %w", err)
		}
		minSize = size
	}

	// Start scanning
	fmt.Println()
	fmt.Println(ui.SectionHeader("Installer Cleanup", 50))
	fmt.Println()

	spinner := ui.NewInlineSpinner()
	spinner.Start("Scanning for installer files...")

	// Scan for installers
	files, err := installer.ScanInstallers(minAge, minSize)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	spinner.Stop(fmt.Sprintf("Found %d installer files", len(files)))

	if len(files) == 0 {
		fmt.Println()
		fmt.Println(ui.SuccessStyle().Render(fmt.Sprintf("  %s No installer files found!", ui.IconCheck)))
		fmt.Println()
		return nil
	}

	// Convert to selector items
	items := installerFilesToSelectorItems(files)

	// Show selector
	selected, err := ui.RunSelector(items, "Select installer files to delete:")
	if err != nil {
		return fmt.Errorf("selector error: %w", err)
	}

	if selected == nil || len(selected) == 0 {
		fmt.Println()
		fmt.Println(ui.MutedStyle().Render("  No files selected. Exiting."))
		fmt.Println()
		return nil
	}

	// Convert back to installer files
	selectedFiles := make([]installer.InstallerFile, 0, len(selected))
	for _, item := range selected {
		// Find the file by path
		for _, file := range files {
			if file.Path == item.Value {
				selectedFiles = append(selectedFiles, file)
				break
			}
		}
	}

	// Show summary
	fmt.Println()
	totalSize := installer.GetTotalSize(selectedFiles)
	fmt.Printf("  %s\n", ui.BoldStyle().Render(fmt.Sprintf("Will delete %d files (%s)",
		len(selectedFiles), core.FormatSize(totalSize))))
	fmt.Println()

	// Confirm
	if !dryRun {
		confirmed, err := ui.Confirm("Proceed with deletion?")
		if err != nil {
			return fmt.Errorf("error: %w", err)
		}
		if !confirmed {
			fmt.Println()
			fmt.Println(ui.MutedStyle().Render("  Cancelled."))
			fmt.Println()
			return nil
		}
	}

	// Delete
	fmt.Println()
	freed, count, cleanErr := installer.CleanInstallers(selectedFiles, dryRun)

	if dryRun {
		fmt.Println()
		fmt.Println(ui.InfoStyle().Render("  [DRY RUN] No files were deleted"))
		fmt.Printf("  Would free: %s from %d files\n", core.FormatSize(freed), count)
		fmt.Println()
	} else {
		fmt.Println()
		if cleanErr != nil {
			fmt.Printf("%s Completed with errors: %v\n", ui.WarningStyle().Render(ui.IconWarning), cleanErr)
		} else {
			fmt.Printf("%s Success!\n", ui.SuccessStyle().Render(ui.IconSuccess))
		}
		fmt.Printf("  Freed: %s from %d files\n", ui.SuccessStyle().Render(core.FormatSize(freed)), count)
		fmt.Println()
	}

	return nil
}

// installerFilesToSelectorItems converts installer files to selector items.
func installerFilesToSelectorItems(files []installer.InstallerFile) []ui.SelectorItem {
	// Group by source
	sourceGroups := installer.GroupBySource(files)

	// Sort sources
	sources := make([]string, 0, len(sourceGroups))
	for s := range sourceGroups {
		sources = append(sources, s)
	}
	sort.Strings(sources)

	// Build items
	items := make([]ui.SelectorItem, 0, len(files))
	for _, source := range sources {
		group := sourceGroups[source]
		// Sort by size descending
		sort.Slice(group, func(i, j int) bool {
			return group[i].Size > group[j].Size
		})

		for _, file := range group {
			// Age
			age := time.Since(file.ModTime)
			ageStr := util.FormatDuration(age)

			item := ui.SelectorItem{
				Label:       file.Name,
				Description: fmt.Sprintf("%s • %s old", file.Path, ageStr),
				Value:       file.Path,
				Size:        core.FormatSize(file.Size),
				Selected:    true,
				Disabled:    false,
				Category:    source,
			}

			items = append(items, item)
		}
	}

	return items
}
